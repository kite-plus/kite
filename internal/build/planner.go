package build

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
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
	prev, next := neighbors(all)
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
			Kind: render.KindSingle,
			URL:  link,
			Path: b.opts.Resolver.OutputPath(link),
			Item: item,
			Prev: prev[i],
			Next: next[i],
			Type: string(item.Kind),
		})
	}
	return nil
}

// neighbors finds, for every item, the one published before it and the one
// after it among the items of the same kind and locale. The input is newest
// first, so the older neighbor is the one that follows.
func neighbors(all []content.Summary) (prev, next []*content.Summary) {
	type run struct {
		kind   content.Kind
		locale string
	}
	prev = make([]*content.Summary, len(all))
	next = make([]*content.Summary, len(all))

	newer := make(map[run]int)
	for i, s := range all {
		key := run{kind: s.Kind, locale: s.Locale}
		if j, ok := newer[key]; ok {
			next[i] = &all[j]
			prev[j] = &all[i]
		}
		newer[key] = i
	}
	return prev, next
}

func (b *Builder) planHome(p *Plan, all []content.Summary) {
	posts := filterKind(all, "post")
	b.paginate(p, render.KindHome, "/", "post", "", "", posts)
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
		b.paginate(p, render.KindList, base, string(t.Kind), "", displayName(t.Dir), items)
	}
}

// planTaxonomies adds one listing per taxonomy and one per term.
//
// Every item the build includes is already loaded, newest first and with its
// terms, which is exactly what a query per term would return. Grouping them
// here rather than asking again term by term took a server's replan after an
// edit on a site with many tags from most of half a second to a fraction.
func (b *Builder) planTaxonomies(p *Plan, all []content.Summary) {
	for _, taxonomy := range b.opts.Types.TaxonomyNames() {
		byTerm := make(map[string][]content.Summary)
		for _, s := range all {
			for _, term := range s.Taxonomies[taxonomy] {
				if listed := byTerm[term]; len(listed) > 0 && listed[len(listed)-1].ID == s.ID {
					continue // named twice by the same item
				}
				byTerm[term] = append(byTerm[term], s)
			}
		}
		if len(byTerm) == 0 {
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

		// Most used first, then by name, the order the index counts them in.
		terms := slices.SortedFunc(maps.Keys(byTerm), func(x, y string) int {
			return cmp.Or(cmp.Compare(len(byTerm[y]), len(byTerm[x])), cmp.Compare(x, y))
		})
		for _, term := range terms {
			base := b.opts.Resolver.ForTerm(taxonomy, term, b.opts.Site.Language)
			// A term keeps the spelling its author used; only Kite's own
			// names are presented.
			b.paginate(p, render.KindTerm, base, taxonomy, term, term, byTerm[term])
		}
	}
}

func (b *Builder) planNotFound(p *Plan) {
	if !b.opts.Engine.HasTemplate(theme.Target{Kind: string(render.KindNotFound), Format: theme.FormatHTML.Name}) {
		return
	}
	p.Targets = append(p.Targets, Target{
		Kind:  render.KindNotFound,
		URL:   "/404",
		Path:  "404.html",
		Title: "Not found",
	})
}

// paginate appends one target per page of a listing.
//
// Every page of a listing is its own target with its own cache key, rather
// than a by-product of rendering the first one. That is what makes paginated
// sections eligible for incremental rebuilds later.
func (b *Builder) paginate(p *Plan, kind render.Kind, base, typ, term, title string, items []content.Summary) {
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
