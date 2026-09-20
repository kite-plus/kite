package build

import (
	"context"
	"fmt"

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
	if err := b.planTaxonomies(ctx, p); err != nil {
		return nil, err
	}
	b.planNotFound(p)

	p.sort()
	return p, nil
}

// loadAll reads every item the build includes, newest first.
func (b *Builder) loadAll(ctx context.Context) ([]content.Summary, error) {
	var out []content.Summary
	cursor := ""
	for {
		page, err := b.opts.Reader.Query(ctx, content.Query{
			Statuses: b.statuses(),
			Limit:    content.MaxLimit,
			Cursor:   cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("build: load content: %w", err)
		}
		out = append(out, page.Items...)
		if !page.HasMore {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

func (b *Builder) planSingles(ctx context.Context, p *Plan, all []content.Summary) error {
	for _, s := range all {
		item, err := b.opts.Reader.Get(ctx, s.ID)
		if err != nil {
			return fmt.Errorf("build: load %s: %w", s.ID, err)
		}
		link := b.opts.Resolver.For(item)
		p.Targets = append(p.Targets, Target{
			Kind: render.KindSingle,
			URL:  link,
			Path: b.opts.Resolver.OutputPath(link),
			Item: item,
			Type: string(item.Kind),
		})
	}
	return nil
}

func (b *Builder) planHome(p *Plan, all []content.Summary) {
	posts := filterKind(all, "post")
	b.paginate(p, render.KindHome, "/", "post", "", posts)
}

func (b *Builder) planLists(p *Plan, all []content.Summary) {
	for _, t := range b.opts.Types.Types() {
		items := filterKind(all, t.Kind)
		if len(items) == 0 {
			continue
		}
		base := b.opts.Resolver.ForList(t.Kind, b.opts.Site.Language)
		if base == "/" {
			continue // the home page already covers this listing
		}
		b.paginate(p, render.KindList, base, string(t.Kind), "", items)
	}
}

// planTaxonomies adds one listing per taxonomy and one per term.
func (b *Builder) planTaxonomies(ctx context.Context, p *Plan) error {
	for _, taxonomy := range b.opts.Types.TaxonomyNames() {
		counts, err := b.opts.Reader.CountTerms(ctx, taxonomy, content.Query{Statuses: b.statuses()})
		if err != nil {
			return fmt.Errorf("build: count %s terms: %w", taxonomy, err)
		}
		if len(counts) == 0 {
			continue
		}

		link := b.opts.Resolver.ForTaxonomy(taxonomy, b.opts.Site.Language)
		p.Targets = append(p.Targets, Target{
			Kind: render.KindTaxonomy,
			URL:  link,
			Path: b.opts.Resolver.OutputPath(link),
			Type: taxonomy,
		})

		for _, c := range counts {
			items, err := b.itemsWithTerm(ctx, taxonomy, c.Term)
			if err != nil {
				return err
			}
			base := b.opts.Resolver.ForTerm(taxonomy, c.Term, b.opts.Site.Language)
			b.paginate(p, render.KindTerm, base, taxonomy, c.Term, items)
		}
	}
	return nil
}

func (b *Builder) itemsWithTerm(ctx context.Context, taxonomy, term string) ([]content.Summary, error) {
	var out []content.Summary
	cursor := ""
	for {
		page, err := b.opts.Reader.Query(ctx, content.Query{
			Statuses: b.statuses(),
			TermsAny: map[string][]string{taxonomy: {term}},
			Limit:    content.MaxLimit,
			Cursor:   cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("build: load %s/%s: %w", taxonomy, term, err)
		}
		out = append(out, page.Items...)
		if !page.HasMore {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

func (b *Builder) planNotFound(p *Plan) {
	if !b.opts.Engine.HasTemplate(theme.Target{Kind: string(render.KindNotFound), Format: theme.FormatHTML.Name}) {
		return
	}
	p.Targets = append(p.Targets, Target{
		Kind: render.KindNotFound,
		URL:  "/404",
		Path: "404.html",
	})
}

// paginate appends one target per page of a listing.
//
// Every page of a listing is its own target with its own cache key, rather
// than a by-product of rendering the first one. That is what makes paginated
// sections eligible for incremental rebuilds later.
func (b *Builder) paginate(p *Plan, kind render.Kind, base, typ, term string, items []content.Summary) {
	size := b.opts.PageSize
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
			Page:       n,
			Items:      items[lo:hi],
			TotalItems: len(items),
		})
	}
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
