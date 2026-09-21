package api

import (
	"net/http"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/publish"
)

// handlePublishState reports how far the content has got.
func (s *Server) handlePublishState(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Publisher == nil {
		writeJSON(w, http.StatusOK, publish.DeliveryState{
			Local:     publish.StepDone,
			Committed: publish.StepNotApplicable,
			Pushed:    publish.StepNotApplicable,
			Deployed:  publish.StepNotApplicable,
		})
		return
	}

	state, err := view.Publisher.State(r.Context())
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// handlePreflight reports what a publish would do, changing nothing.
func (s *Server) handlePreflight(w http.ResponseWriter, r *http.Request) {
	view, req, ok := s.publishRequest(w, r)
	if !ok {
		return
	}
	plan, err := view.Publisher.Preflight(r.Context(), req)
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// handlePublish preflights and then carries out the plan.
//
// The two are done together here rather than asking a client to hold a plan
// between requests: a plan describes a moment, and a repository can change
// between the moment it was described and the moment it is acted on.
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	view, req, ok := s.publishRequest(w, r)
	if !ok {
		return
	}

	plan, err := view.Publisher.Preflight(r.Context(), req)
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(plan.Problems) > 0 {
		writeJSON(w, http.StatusConflict, PublishRefused{
			Error: ErrorDetail{
				Code:    CodePublishRefused,
				Message: plan.Problems[0].Detail,
			},
			Plan: plan,
		})
		return
	}
	// Warnings are shown once and then acknowledged, so that "you have staged
	// your own version of this file" is read rather than clicked past.
	if len(plan.Warnings) > 0 && !req.Force {
		writeJSON(w, http.StatusConflict, PublishRefused{
			Error: ErrorDetail{
				Code:    CodePublishNeedsConfirmation,
				Message: plan.Warnings[0].Detail,
			},
			Plan: plan,
		})
		return
	}

	result, err := view.Publisher.Apply(r.Context(), plan)
	if err != nil {
		// A push can fail after the commit succeeded. Reporting only the
		// error would invite an author to repeat a commit they already have,
		// so what did happen is reported alongside what did not.
		writeJSON(w, http.StatusConflict, PublishRefused{
			Error: ErrorDetail{Code: CodePublishFailed, Message: err.Error()},
			Plan:  plan,
			Done:  result,
		})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// publishRequest reads a publish body and resolves the items it names.
func (s *Server) publishRequest(w http.ResponseWriter, r *http.Request) (View, publish.Request, bool) {
	view := s.src()

	if view.Publisher == nil {
		fail(w, http.StatusNotImplemented, CodeReadOnly,
			"this project has no publisher configured")
		return view, publish.Request{}, false
	}
	if !s.writable(w, view) {
		return view, publish.Request{}, false
	}

	body, ok := decodeJSON[PublishBody](s, w, r)
	if !ok {
		return view, publish.Request{}, false
	}

	req := publish.Request{
		Paths:   body.Paths,
		Message: body.Message,
		Push:    body.Push,
		Force:   body.Force,
	}
	// An item is named by id; where its bytes live is the store's business,
	// not the caller's.
	var titles []string
	for _, id := range body.IDs {
		item, err := view.Reader.Get(r.Context(), content.ID(id))
		if err != nil {
			s.failErr(w, err)
			return view, publish.Request{}, false
		}
		req.Paths = append(req.Paths, string(item.Locator))
		titles = append(titles, item.Title)
	}
	// A subject naming the post reads better in a log than one naming a
	// directory, and only the caller knows the title.
	if req.Message == "" && len(titles) == 1 && len(body.Paths) == 0 {
		req.Message = "publish: " + titles[0]
	}
	if len(req.Paths) == 0 {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "ids",
			"name what to publish, with ids or paths")
		return view, publish.Request{}, false
	}
	return view, req, true
}
