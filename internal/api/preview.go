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

	item := draftItem(r, view, draft)
	if item.Slug == "" {
		item.Slug = "preview"
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

// handleWordCount counts an unsaved draft's words as its page will.
//
// The count is made here for the reason the preview is: an editor counting
// the markdown itself would be a second reading of it, and would show a
// number the site does not.
func (s *Server) handleWordCount(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.WordCount == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot count words")
		return
	}

	draft, ok := s.decodeDraft(w, r)
	if !ok {
		return
	}
	words, err := view.WordCount(r.Context(), draftItem(r, view, draft))
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, WordCount{Words: words})
}

// draftItem is the item a draft will be once saved. An id, when the client
// has one, keeps it in the bundle it already lives in, which is what lets its
// own images resolve.
func draftItem(r *http.Request, view View, draft Draft) *content.Content {
	item := draft.contentOf(content.ID(r.URL.Query().Get("id")))
	if existing, err := view.Reader.Get(r.Context(), item.ID); err == nil {
		item.Locator = existing.Locator
		item.CreatedAt = existing.CreatedAt
	}
	return item
}
