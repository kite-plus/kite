package api

import (
	"net/http"
	"strconv"

	"github.com/kite-plus/kite/internal/content"
)

// handlePreview renders an unsaved draft through the real theme.
//
// The rendering happens here and not in the browser. A front end that ran its
// own markdown library would disagree with the build about footnotes, code
// highlighting, raw html and every extension either side adds later, and
// "the preview does not match the site" is a complaint with no end. The
// preview is the same renderer, the same theme and the same resolver a build
// uses, applied to bytes that are not on disk yet.
func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Preview == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot render previews")
		return
	}

	draft, ok := s.decodeDraft(w, r)
	if !ok {
		return
	}
	if draft.Kind == "" || view.Types.Get(content.Kind(draft.Kind)) == nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "kind",
			"a preview needs a known kind to choose a template")
		return
	}

	// An id, when the client has one, is what lets a draft's own images
	// resolve: the item keeps the bundle it already lives in.
	item := draft.contentOf(content.ID(r.URL.Query().Get("id")))
	if item.Slug == "" {
		item.Slug = "preview"
	}
	if existing, err := view.Reader.Get(r.Context(), item.ID); err == nil {
		item.Locator = existing.Locator
		item.CreatedAt = existing.CreatedAt
	}

	html, err := view.Preview(r.Context(), item)
	if err != nil {
		s.failErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(html)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
}
