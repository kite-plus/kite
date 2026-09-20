package build

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/markdown"
	"github.com/kite-plus/kite/internal/render/theme"
	kurl "github.com/kite-plus/kite/internal/render/url"
)

// Options configures a build.
type Options struct {
	Site     render.SiteInfo
	Reader   content.Reader
	Resolver *kurl.Resolver
	Engine   *theme.Engine
	Markdown *markdown.Renderer
	Hooks    *hook.Bus
	Types    *content.Registry
	Emitter  *Emitter

	// PageSize is how many items a listing shows.
	PageSize int

	// IncludeDrafts renders unpublished items, for local preview.
	IncludeDrafts bool

	// Now freezes the build clock. Zero means time.Now at the start.
	Now time.Time
}

// Stats reports what a build did.
type Stats struct {
	Targets  int           `json:"targets"`
	Rendered int           `json:"rendered"`
	Skipped  int           `json:"skipped"`
	Extra    int           `json:"extra"`
	Duration time.Duration `json:"-"`
}

// Builder renders a site.
type Builder struct {
	opts     Options
	buildCtx *Context
	site     render.Site
}

// New returns a builder.
func New(opts Options) (*Builder, error) {
	switch {
	case opts.Reader == nil:
		return nil, fmt.Errorf("build: a reader is required")
	case opts.Resolver == nil:
		return nil, fmt.Errorf("build: a url resolver is required")
	case opts.Engine == nil:
		return nil, fmt.Errorf("build: a theme engine is required")
	}
	if opts.Markdown == nil {
		opts.Markdown = markdown.New(markdown.DefaultOptions())
	}
	if opts.Hooks == nil {
		opts.Hooks = hook.NewBus()
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	b := &Builder{opts: opts}
	b.buildCtx = NewContext(opts.Now, b.sharedKey()...)
	b.site = b.newSite(b.buildCtx)
	return b, nil
}

// Plan expands the site into every output it would produce.
//
// A server resolves a request by looking up the target the build would have
// written for that URL, which is what keeps the two runtimes from drifting:
// they render the same target through the same code.
func (b *Builder) Plan(ctx context.Context) (*Plan, error) {
	return b.plan(ctx, b.buildCtx)
}

// Site is the site view handed to templates.
func (b *Builder) Site() render.Site { return b.site }

// Render produces one output. Request is nil during a build; a server passes
// the live request, which templates reach only through {{ with .Request }}.
func (b *Builder) Render(ctx context.Context, t Target, req render.Request) ([]byte, hook.PageInfo, error) {
	return b.renderTarget(ctx, b.buildCtx.ForOutput(), t, req)
}

// Run plans and renders the whole site.
func (b *Builder) Run(ctx context.Context) (Stats, error) {
	start := time.Now()
	var stats Stats

	if b.opts.Emitter == nil {
		return stats, fmt.Errorf("build: an emitter is required to write a site")
	}

	plan, err := b.Plan(ctx)
	if err != nil {
		return stats, err
	}
	stats.Targets = plan.Len()

	var pages []hook.PageInfo

	// The loop is per output target, with the skip check in place from the
	// start. v1 never skips; making that decision real later is a change to
	// one condition rather than to the shape of the build.
	for _, target := range plan.Targets {
		if err := ctx.Err(); err != nil {
			return stats, err
		}

		out := b.buildCtx.ForOutput()
		if b.cached(out, target) {
			stats.Skipped++
			continue
		}

		html, info, err := b.renderTarget(ctx, out, target, nil)
		if err != nil {
			return stats, err
		}
		if err := b.opts.Emitter.Write(target.Path, html); err != nil {
			return stats, err
		}
		stats.Rendered++
		pages = append(pages, info)

		if err := b.opts.Hooks.PageRendered(ctx, &info); err != nil {
			return stats, err
		}
	}

	extra, err := b.runCompletionHooks(ctx, pages)
	if err != nil {
		return stats, err
	}
	stats.Extra = extra

	if err := b.opts.Emitter.Commit(); err != nil {
		return stats, err
	}
	stats.Duration = time.Since(start)
	return stats, nil
}

// cached reports whether a target's previous output is still valid.
//
// v1 always rebuilds. The call sits here so that the dependency recording,
// the cache key and the loop shape are all exercised from the first release,
// leaving only the lookup itself to add later.
func (b *Builder) cached(*Context, Target) bool { return false }

func (b *Builder) sharedKey() []string {
	return []string{
		"site=" + b.opts.Site.BaseURL + "|" + b.opts.Site.Title + "|" + b.opts.Site.Language,
		"hooks=" + string(b.opts.Hooks.CacheKey()),
	}
}

func (b *Builder) newSite(c *Context) render.Site {
	info := b.opts.Site
	info.BuildTime = c.Now()
	info.Build = true
	if info.Taxonomies == nil {
		info.Taxonomies = b.opts.Types.TaxonomyNames()
	}
	return render.NewSite(info)
}

// statuses returns the statuses a build includes.
func (b *Builder) statuses() []content.Status {
	if b.opts.IncludeDrafts {
		return nil
	}
	return []content.Status{content.StatusPublished, content.StatusScheduled}
}

// renderTarget produces the bytes of one output.
func (b *Builder) renderTarget(ctx context.Context, out *Context, t Target, req render.Request) ([]byte, hook.PageInfo, error) {
	var info hook.PageInfo

	page, pages, err := b.pages(ctx, out, t)
	if err != nil {
		return nil, info, err
	}

	target := theme.Target{
		Kind:   string(t.Kind),
		Type:   t.Type,
		Layout: t.Layout,
		Format: theme.FormatHTML.Name,
	}
	found, _, ok := b.opts.Engine.Lookup(target)
	if !ok {
		return nil, info, fmt.Errorf("build: %s: no template found", t.ID())
	}
	out.Read(Node{Kind: NodeTemplate, ID: found.Path}, "", "content")

	var paginator render.Paginator
	if t.Kind != render.KindSingle {
		paginator = render.NewPaginator(b.listBase(t), max(1, t.Page), b.opts.PageSize, t.TotalItems, b.opts.Resolver)
	}

	data := render.NewContext(render.ContextOptions{
		Site:      b.site,
		Page:      page,
		Pages:     pages,
		Paginator: paginator,
		Terms:     b.terms(ctx, out, t),
		Request:   req, // nil in a build; themes guard with {{ with .Request }}
	})

	html, err := b.opts.Engine.Render(target, data)
	if err != nil {
		return nil, info, fmt.Errorf("build: %s: %w", t.ID(), err)
	}

	doc := hook.HTMLDoc{Item: t.Item, URL: t.URL, HTML: string(html)}
	if err := b.opts.Hooks.TransformHTML(ctx, &doc); err != nil {
		return nil, info, err
	}

	info = hook.PageInfo{
		Item:       t.Item,
		URL:        t.URL,
		OutputPath: t.Path,
		Indexable:  t.Kind != render.KindNotFound,
	}
	if page != nil {
		info.Title = page.Title()
		info.Excerpt = page.Excerpt()
	}
	return []byte(doc.HTML), info, nil
}

// pages builds the Page view of a target and of everything it lists.
func (b *Builder) pages(ctx context.Context, out *Context, t Target) (render.Page, []render.Page, error) {
	var page render.Page

	if t.Item != nil {
		doc, err := b.renderBody(ctx, t.Item)
		if err != nil {
			return nil, nil, err
		}
		out.Read(Node{Kind: NodeContent, ID: string(t.Item.ID)}, string(t.Item.Revision),
			"title", "slug", "body", "params", "published_at", "updated_at", "taxonomies")
		page = render.NewPage(t.Item, render.PageOptions{
			Kind:     t.Kind,
			Rendered: doc,
			Resolver: b.opts.Resolver,
			Terms:    b.termsOf(t.Item),
		})
	} else if t.Kind != render.KindSingle {
		page = b.listingPage(t)
	}

	listed := make([]render.Page, 0, len(t.Items))
	for _, s := range t.Items {
		// A listing reads only this projection of each item, so editing a body
		// does not invalidate the pages that merely link to it.
		out.Read(Node{Kind: NodeContent, ID: string(s.ID)}, string(s.Revision),
			"title", "slug", "excerpt", "published_at", "taxonomies")
		listed = append(listed, render.NewPage(summaryToContent(s), render.PageOptions{
			Kind:     render.KindSingle,
			Rendered: &markdown.Document{Excerpt: s.Excerpt},
			Resolver: b.opts.Resolver,
			Terms:    b.termsOfMap(s.Taxonomies),
		}))
	}
	return page, listed, nil
}

// renderBody runs the markdown pipeline with the markdown hooks around it.
func (b *Builder) renderBody(ctx context.Context, item *content.Content) (*markdown.Document, error) {
	src := hook.MarkdownDoc{Item: item, Source: item.Body.Raw}
	if err := b.opts.Hooks.TransformMarkdown(ctx, &src); err != nil {
		return nil, err
	}
	doc, err := b.opts.Markdown.Render(src.Source)
	if err != nil {
		return nil, fmt.Errorf("build: render %s: %w", item.ID, err)
	}
	return doc, nil
}

// listingPage synthesizes the Page of a listing, so that a theme can write
// {{ .Page.Title }} on every kind of page.
func (b *Builder) listingPage(t Target) render.Page {
	title := t.Term
	if title == "" {
		if ct := b.opts.Types.Get(content.Kind(t.Type)); ct != nil {
			title = ct.Label
		} else {
			title = t.Type
		}
	}
	item := &content.Content{
		ID:    content.ID("listing:" + t.URL),
		Kind:  content.Kind(t.Type),
		Slug:  t.URL,
		Title: title,
	}
	return render.NewPage(item, render.PageOptions{Kind: t.Kind, Rendered: &markdown.Document{}})
}

func (b *Builder) listBase(t Target) string {
	switch t.Kind {
	case render.KindTerm:
		return b.opts.Resolver.ForTerm(t.Type, t.Term, b.opts.Site.Language)
	case render.KindTaxonomy:
		return b.opts.Resolver.ForTaxonomy(t.Type, b.opts.Site.Language)
	case render.KindHome:
		return "/"
	default:
		return b.opts.Resolver.ForList(content.Kind(t.Type), b.opts.Site.Language)
	}
}

func (b *Builder) terms(ctx context.Context, out *Context, t Target) []render.Term {
	if t.Kind != render.KindTaxonomy {
		return nil
	}
	counts, err := b.opts.Reader.CountTerms(ctx, t.Type, content.Query{Statuses: b.statuses()})
	if err != nil {
		return nil
	}
	out.Read(Node{Kind: NodeTaxonomy, ID: t.Type}, "", "terms")

	terms := make([]render.Term, 0, len(counts))
	for _, c := range counts {
		terms = append(terms, render.NewTerm(c.Taxonomy, c.Term, c.Count, b.opts.Resolver, b.opts.Site.Language, nil))
	}
	return terms
}

func (b *Builder) termsOf(item *content.Content) map[string][]render.Term {
	return b.termsOfMap(item.Taxonomies)
}

func (b *Builder) termsOfMap(taxonomies map[string][]string) map[string][]render.Term {
	if len(taxonomies) == 0 {
		return nil
	}
	out := make(map[string][]render.Term, len(taxonomies))
	for _, name := range slices.Sorted(maps.Keys(taxonomies)) {
		for _, term := range taxonomies[name] {
			out[name] = append(out[name],
				render.NewTerm(name, term, 0, b.opts.Resolver, b.opts.Site.Language, nil))
		}
	}
	return out
}

func (b *Builder) runCompletionHooks(ctx context.Context, pages []hook.PageInfo) (int, error) {
	var extra int
	info := hook.BuildInfo{
		Site: hook.SiteInfo{
			Title:       b.opts.Site.Title,
			Description: b.opts.Site.Description,
			BaseURL:     b.opts.Site.BaseURL,
			Language:    b.opts.Site.Language,
		},
		Pages: pages,
		Emit: func(path string, data []byte) error {
			extra++
			return b.opts.Emitter.Write(path, data)
		},
	}
	if err := b.opts.Hooks.BuildComplete(ctx, &info); err != nil {
		return extra, err
	}
	return extra, nil
}

// summaryToContent lifts a list projection into the shape a Page needs. Only
// the fields a listing may read are populated, which keeps a theme from
// reaching past the projection its dependency record claims.
func summaryToContent(s content.Summary) *content.Content {
	return &content.Content{
		ID:          s.ID,
		Kind:        s.Kind,
		Slug:        s.Slug,
		Title:       s.Title,
		Status:      s.Status,
		Taxonomies:  s.Taxonomies,
		Locale:      s.Locale,
		Locator:     s.Locator,
		Revision:    s.Revision,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		PublishedAt: s.PublishedAt,
	}
}
