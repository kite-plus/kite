package api

import (
	"net/http"
)

// handleOpenPreview draws the site with a theme or settings being tried, at
// an address of its own that nothing but the admin links to.
func (s *Server) handleOpenPreview(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Previews == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot draw previews")
		return
	}
	body, ok := decodeJSON[PreviewBody](s, w, r)
	if !ok {
		return
	}
	trial, ok := trialOf(w, view, body)
	if !ok {
		return
	}
	token, err := view.Previews.Open(r.Context(), trial, previewPrefix(r))
	if err != nil {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, Preview{Token: token, URL: previewURL(token)})
}

// handleUpdatePreview draws an open preview again, keeping its address, so a
// page showing it only has to reload.
func (s *Server) handleUpdatePreview(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Previews == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot draw previews")
		return
	}
	body, ok := decodeJSON[PreviewBody](s, w, r)
	if !ok {
		return
	}
	trial, ok := trialOf(w, view, body)
	if !ok {
		return
	}
	token := r.PathValue("token")
	open, err := view.Previews.Update(r.Context(), token, trial)
	switch {
	case !open:
		fail(w, http.StatusNotFound, CodeNotFound, "no such preview; it may have been left unused too long")
	case err != nil:
		fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
	default:
		writeJSON(w, http.StatusOK, Preview{Token: token, URL: previewURL(token)})
	}
}

// handleClosePreview forgets a preview.
func (s *Server) handleClosePreview(w http.ResponseWriter, r *http.Request) {
	if view := s.src(); view.Previews != nil {
		view.Previews.Close(r.PathValue("token"))
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePreviewPage answers for a page or a file of a preview.
func (s *Server) handlePreviewPage(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Previews == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot draw previews")
		return
	}
	if !view.Previews.Serve(w, r, r.PathValue("token"), Prefix+r.URL.Path) {
		fail(w, http.StatusNotFound, CodeNotFound, "no such preview; it may have been left unused too long")
	}
}

// trialOf reads what a preview is to draw, answering for itself when the
// theme it names cannot be drawn.
func trialOf(w http.ResponseWriter, view View, body PreviewBody) (Trial, bool) {
	trial := Trial(body)
	if trial.Theme == "" {
		trial.Theme = view.Theme
	}
	if _, problem := usableTheme(view, trial.Theme); problem != "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "theme", problem)
		return Trial{}, false
	}
	if trial.Settings == nil {
		trial.Settings = view.ThemeSettings
	}
	return trial, true
}

// previewPrefix is the absolute address previews are served under, for the
// links a preview's pages make to carry.
func previewPrefix(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + Prefix + "/previews/"
}

func previewURL(token string) string { return Prefix + "/previews/" + token + "/" }
