package build

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/theme"
)

// plan expands the site into the complete list of files to produce.
func (b *Builder) plan(ctx context.Context, c *Context) (*Plan, error) {
	p := &Plan{}

	all, err := b.loadAll(ctx)
	if err != nil {
		return nil, err
	}
	c.Read(Node{Kind: NodeConfig, ID: "site"}, "", "title", "baseURL", "language")

	if err := b.planSingles(ctx, p, all); err != nil {
		return nil, err
	}
	b.planHome(p, all)
	b.planLists(p, all)
	b.planTaxonomies(p, all)
	b.planNotFound(p)
	if err := distinctPages(p); err != nil {
		return nil, err
	}
	if err := b.planAliases(p); err != nil {
		return nil, err
	}

	p.sort()
	return p, nil
}

// loadAll reads every item the build includes, newest first.
func (b *Builder) loadAll(ctx context.Context) ([]content.Summary, error) {
	var out []content.Summary
	q := b.scope()
	q.Limit = content.MaxLimit
	for {
		page, err := b.opts.Reader.Query(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("build: load content: %w", err)
		}
		out = append(out, page.Items...)
		if !page.HasMore {
			return out, nil
		}
		q.Cursor = page.NextCursor
	}
}

func (b *Builder) planSingles(ctx context.Context, p *Plan, all []content.Summary) error {
	prev, next := b.neighbors(all)
	ids := make([]content.ID, len(all))
	for i, s := range all {
		ids[i] = s.ID
	}
	items, err := b.opts.Reader.GetMany(ctx, ids)
	if err != nil {
		return fmt.Errorf("build: load content: %w", err)
	}
	if len(items) != len(all) {
		return fmt.Errorf("build: %d item(s) disappeared while the site was being planned", len(all)-len(items))
	}
	for i, item := range items {
		link := b.opts.Resolver.For(item)
		p.Targets = append(p.Targets, Target{
			Kind:   render.KindSingle,
			URL:    link,
			Path:   b.opts.Resolver.OutputPath(link),
			Item:   item,
			Prev:   prev[i],
			Next:   next[i],
			Type:   string(item.Kind),
			Layout: LayoutOf(item),
		})
	}
	return nil
}

// distinctPages refuses an item whose address another page already has, as
// a page whose slug is posts or tags/go would: a host serves one file there,
// and the other page would be lost with nothing said.
func distinctPages(p *Plan) error {
	seen := make(map[string]Target, len(p.Targets))
	for _, t := range p.Targets {
		if t.Kind != render.KindSingle && t.Kind != render.KindNotFound {
			seen[Address(t.URL)] = t
		}
	}
	for _, t := range p.Targets {
		if t.Kind != render.KindSingle {
			continue
		}
		key := Address(t.URL)
		if held, ok := seen[key]; ok {
			return fmt.Errorf("build: %s is at %s, which is %s's address too; give it another slug",
				describeTarget(t), t.URL, describeTarget(held))
		}
		seen[key] = t
	}
	return nil
}

// LayoutOf is the layout an item asks for in its front matter, as in Hugo:
// layout: links. A value that could not be a template's name is ignored, and
// the item is drawn with its type's own template, as it is when the layout it
// names has no template.
func LayoutOf(item *content.Content) string {
	name, _ := item.Meta["layout"].(string)
	if !theme.ValidLayoutName(name) {
		return ""
	}
	return name
}

// neighbors finds, for every item, the one before it and the one after it in
// the order its kind is read in, among the items of the same kind and locale:
// by date, where the one before is the older one, or by weight. The input is
// newest first.
func (b *Builder) neighbors(all []content.Summary) (prev, next []*content.Summary) {
	type run struct {
		kind   content.Kind
		locale string
	}
	runs := make(map[run][]int)
	for i, s := range all {
		key := run{kind: s.Kind, locale: s.Locale}
		runs[key] = append(runs[key], i)
	}
	prev = make([]*content.Summary, len(all))
	next = make([]*content.Summary, len(all))
	for key, read := range runs {
		if b.readsByWeight(key.kind) {
			slices.SortStableFunc(read, func(x, y int) int { return byWeight(all[x], all[y]) })
		} else {
			slices.Reverse(read) // oldest first
		}
		for j, i := range read {
			if j > 0 {
				prev[i] = &all[read[j-1]]
			}
			if j+1 < len(read) {
				next[i] = &all[read[j+1]]
			}
		}
	}
	return prev, next
}

// readsByWeight reports whether a kind is listed and read by weight.
func (b *Builder) readsByWeight(kind content.Kind) bool {
	t := b.opts.Types.Get(kind)
	return t != nil && t.Order == content.OrderWeight
}

// byWeight orders items by the weight their front matter gives, smallest
// first, and then those that give none, or 0, as Hugo does; either by title
// where they tie.
func byWeight(a, b content.Summary) int {
	x, weighed := weightOf(a)
	y, weighs := weightOf(b)
	switch {
	case weighed != weighs && weighed:
		return -1
	case weighed != weighs:
		return 1
	case x != y:
		return cmp.Compare(x, y)
	}
	return cmp.Or(strings.Compare(a.Title, b.Title), strings.Compare(string(a.ID), string(b.ID)))
}

func weightOf(s content.Summary) (float64, bool) {
	var w float64
	switch v := s.Meta["weight"].(type) {
	case int:
		w = float64(v)
	case int64:
		w = float64(v)
	case uint64:
		w = float64(v)
	case float64:
		w = v
	}
	return w, w != 0
}

func (b *Builder) planHome(p *Plan, all []content.Summary) {
	posts := filterKind(all, "post")
	b.paginate(p, render.KindHome, b.opts.Resolver.ForHome(b.opts.Site.Language), "post", "", "", posts)
}

func (b *Builder) planLists(p *Plan, all []content.Summary) {
	home := b.opts.Resolver.ForHome(b.opts.Site.Language)
	for _, t := range b.opts.Types.Types() {
		items := filterKind(all, t.Kind)
		if len(items) == 0 {
			continue
		}
		if b.readsByWeight(t.Kind) {
			slices.SortStableFunc(items, byWeight)
		}
		base := b.opts.Resolver.ForList(t.Kind, b.opts.Site.Language)
		if base == home {
			continue // the home page already covers this listing
		}
		b.paginate(p, render.KindList, base, string(t.Kind), "", displayName(t.Dir), items)
	}
}

// planTaxonomies adds one listing per taxonomy and one per term.
//
// Every item the build includes is already loaded, newest first and with its
// terms, which is exactly what a query per term would return. Grouping them
// here rather than asking again term by term took a server's replan after an
// edit on a site with many tags from most of half a second to a fraction.
//
// Terms are grouped by slug, as their addresses are: Go and go are both
// /tags/go/, so they are one term, whose page lists the items of both and is
// called what most of them write. A term with an empty slug would be the
// taxonomy's own listing, and has no page.
func (b *Builder) planTaxonomies(p *Plan, all []content.Summary) {
	type term struct {
		items     []content.Summary
		spellings map[string]int
	}
	for _, taxonomy := range b.opts.Types.TaxonomyNames() {
		bySlug := make(map[string]*term)
		for _, s := range all {
			written := s.Taxonomies[taxonomy]
			for i, name := range written {
				slug := content.TermSlug(name)
				if slug == "" || slices.Contains(written[:i], name) {
					continue // no page of its own, or named twice by the same item
				}
				t := bySlug[slug]
				if t == nil {
					t = &term{spellings: make(map[string]int)}
					bySlug[slug] = t
				}
				t.spellings[name]++
				if n := len(t.items); n > 0 && t.items[n-1].ID == s.ID {
					continue // the same term written another way by the same item
				}
				t.items = append(t.items, s)
			}
		}
		if len(bySlug) == 0 {
			continue
		}

		link := b.opts.Resolver.ForTaxonomy(taxonomy, b.opts.Site.Language)
		p.Targets = append(p.Targets, Target{
			Kind:  render.KindTaxonomy,
			URL:   link,
			Path:  b.opts.Resolver.OutputPath(link),
			Type:  taxonomy,
			Title: displayName(taxonomy),
		})

		for _, slug := range slices.Sorted(maps.Keys(bySlug)) {
			t := bySlug[slug]
			// A term keeps a spelling its authors used; only Kite's own names
			// are presented.
			name := content.TermName(t.spellings)
			base := b.opts.Resolver.ForTerm(taxonomy, name, b.opts.Site.Language)
			b.paginate(p, render.KindTerm, base, taxonomy, name, name, t.items)
		}
	}
}

func (b *Builder) planNotFound(p *Plan) {
	if !b.opts.Engine.HasTemplate(theme.Target{Kind: string(render.KindNotFound), Format: theme.FormatHTML.Name}) {
		return
	}
	p.Targets = append(p.Targets, Target{
		Kind:  render.KindNotFound,
		URL:   b.opts.Resolver.Rel("404"),
		Path:  "404.html",
		Title: "Not found",
	})
}

// paginate appends one target per page of a listing, or a single one for a
// kind of listing that shows every item on one page.
//
// Every page of a listing is its own target with its own cache key, rather
// than a by-product of rendering the first one. That is what makes paginated
// sections eligible for incremental rebuilds later.
func (b *Builder) paginate(p *Plan, kind render.Kind, base, typ, term, title string, items []content.Summary) {
	size := b.pageSize(kind)
	if size == 0 {
		size = max(1, len(items))
	}
	pages := max(1, (len(items)+size-1)/size)

	for n := 1; n <= pages; n++ {
		lo := (n - 1) * size
		hi := min(lo+size, len(items))
		link := b.opts.Resolver.ForPage(base, n)
		p.Targets = append(p.Targets, Target{
			Kind:       kind,
			URL:        link,
			Path:       b.opts.Resolver.OutputPath(link),
			Type:       typ,
			Term:       term,
			Title:      title,
			Page:       n,
			Items:      items[lo:hi],
			TotalItems: len(items),
		})
	}
}

// displayName turns an internal name into one fit to print: "posts" becomes
// "Posts". Only names Kite chose are transformed, never an author's words.
func displayName(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

func filterKind(all []content.Summary, kind content.Kind) []content.Summary {
	var out []content.Summary
	for _, s := range all {
		if s.Kind == kind {
			out = append(out, s)
		}
	}
	return out
}
