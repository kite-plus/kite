package build

import (
	"cmp"
	"slices"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render"
)

// Target is one file the build will produce.
//
// Taxonomy listings and paginated pages are targets in their own right, with
// their own cache keys, from the very first version. A design that treats them
// as a side effect of rendering something else can never make them
// incremental, which is the mistake that leaves other generators unable to do
// partial rebuilds of paginated sections.
type Target struct {
	Kind render.Kind

	// URL is the site-relative address, and Path the file it is written to.
	// Both come from the resolver so they cannot disagree.
	URL  string
	Path string

	// Item is set for single pages.
	Item *content.Content

	// Type is the content type for single and list pages, or the taxonomy name
	// for taxonomy and term pages.
	Type string

	// Term is set for term pages.
	Term string

	// Page is the 1-based pagination index for list-like targets.
	Page int

	// Items are the summaries this target lists.
	Items []content.Summary

	// TotalItems is the size of the full listing this target paginates.
	TotalItems int

	// Layout overrides the template base name.
	Layout string

	// Title is what a listing calls itself. It is set here rather than derived
	// while rendering, because what a page is called is a question about the
	// site, not about the template that happens to draw it.
	Title string
}

// ID is a stable identifier used for cache keys and logs.
func (t Target) ID() string { return string(t.Kind) + " " + t.URL }

// Plan is the ordered set of targets a build will render.
type Plan struct {
	Targets []Target
}

// Len reports how many outputs the plan produces.
func (p *Plan) Len() int { return len(p.Targets) }

// sort orders targets deterministically.
//
// Output must never depend on map iteration order: two builds of the same
// input have to produce the same bytes, or `kite build --verify` is
// meaningless and a reproducible deployment is impossible.
func (p *Plan) sort() {
	slices.SortFunc(p.Targets, func(a, b Target) int {
		return cmp.Or(
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(string(a.Kind), string(b.Kind)),
			cmp.Compare(a.Page, b.Page),
		)
	})
}
