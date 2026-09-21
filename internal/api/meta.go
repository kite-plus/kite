package api

import (
	"net/http"
	"slices"

	"github.com/kite-plus/kite/internal/content"
)

// handleSite describes the open project.
func (s *Server) handleSite(w http.ResponseWriter, r *http.Request) {
	view := s.src()

	info := SiteInfo{
		Title:       view.Site.Title,
		Description: view.Site.Description,
		BaseURL:     view.Site.BaseURL,
		Language:    view.Site.Language,
		Store:       view.Store,
		Runtime:     view.Runtime,
		Theme:       view.Theme,
		Version:     view.Version,
		Problems:    view.Problems,
		Counts:      make(map[string]int),
	}

	for _, t := range view.Types.Types() {
		n, err := view.Reader.Count(r.Context(), content.Query{Kinds: []content.Kind{t.Kind}})
		if err != nil {
			s.failErr(w, err)
			return
		}
		info.Counts[string(t.Kind)] = n
	}
	writeJSON(w, http.StatusOK, info)
}

// handleContentTypes returns the registry, including each type's field schema.
//
// The admin generates its forms from this rather than knowing about posts and
// pages, which is what makes opening custom types later additive.
func (s *Server) handleContentTypes(w http.ResponseWriter, r *http.Request) {
	types := s.src().Types.Types()

	out := make([]ContentType, 0, len(types))
	for _, t := range types {
		out = append(out, contentTypeOf(t))
	}
	writeJSON(w, http.StatusOK, List[ContentType]{Items: out})
}

// handleTaxonomies lists the taxonomies and how many terms each holds.
func (s *Server) handleTaxonomies(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	names := view.Types.TaxonomyNames()

	out := make([]Taxonomy, 0, len(names))
	for _, name := range names {
		counts, err := view.Reader.CountTerms(r.Context(), name, content.Query{})
		if err != nil {
			s.failErr(w, err)
			return
		}
		out = append(out, Taxonomy{
			Name:  name,
			Terms: len(counts),
			URL:   view.Resolver.ForTaxonomy(name, view.Site.Language),
		})
	}
	writeJSON(w, http.StatusOK, List[Taxonomy]{Items: out})
}

// handleTerms aggregates one taxonomy's terms over a filtered set.
func (s *Server) handleTerms(w http.ResponseWriter, r *http.Request) {
	view := s.src()

	taxonomy := r.PathValue("taxonomy")
	if !slices.Contains(view.Types.TaxonomyNames(), taxonomy) {
		fail(w, http.StatusNotFound, CodeNotFound, "no such taxonomy: "+taxonomy)
		return
	}

	q, err := parseQuery(r.URL.Query())
	if err != nil {
		s.failQuery(w, err)
		return
	}

	counts, err := view.Reader.CountTerms(r.Context(), taxonomy, q)
	if err != nil {
		s.failErr(w, err)
		return
	}

	out := make([]TermCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, TermCount{
			Term:  c.Term,
			Count: c.Count,
			URL:   view.Resolver.ForTerm(taxonomy, c.Term, view.Site.Language),
		})
	}
	writeJSON(w, http.StatusOK, List[TermCount]{Items: out})
}
