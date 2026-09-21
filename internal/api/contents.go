package api

import (
	"net/http"
	"strconv"

	"github.com/kite-plus/kite/internal/content"
)

// handleContents lists content.
func (s *Server) handleContents(w http.ResponseWriter, r *http.Request) {
	view := s.src()

	q, err := parseQuery(r.URL.Query())
	if err != nil {
		s.failQuery(w, err)
		return
	}

	page, err := view.Reader.Query(r.Context(), q)
	if err != nil {
		s.failErr(w, err)
		return
	}

	out := List[Summary]{
		Items:      make([]Summary, 0, len(page.Items)),
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}
	for _, item := range page.Items {
		out.Items = append(out.Items, summaryOf(item, view.Resolver))
	}

	// Counting is opt-in: it is a second query over the whole filtered set,
	// and a listing that only wants the next page should not pay for it.
	if wantsCount(r) {
		total, err := view.Reader.Count(r.Context(), q)
		if err != nil {
			s.failErr(w, err)
			return
		}
		out.Total = &total
	}
	writeJSON(w, http.StatusOK, out)
}

// handleContent returns one item in full.
func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	view := s.src()

	item, err := view.Reader.Get(r.Context(), content.ID(r.PathValue("id")))
	if err != nil {
		s.failErr(w, err)
		return
	}
	// The revision travels as an entity tag so an editor can send it straight
	// back in If-Match, rather than having to dig it out of the body.
	w.Header().Set("ETag", etag(item.Revision))
	writeJSON(w, http.StatusOK, itemOf(item, view.Resolver))
}

func wantsCount(r *http.Request) bool {
	ok, err := strconv.ParseBool(r.URL.Query().Get("count"))
	return err == nil && ok
}
