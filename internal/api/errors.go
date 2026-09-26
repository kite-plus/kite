package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kite-plus/kite/internal/content"
)

// Error codes are part of the published contract. A client switches on the
// code; the message is for a human and may be reworded at any time.
const (
	CodeNotFound         = "not_found"
	CodeInvalidRequest   = "invalid_request"
	CodeUnsupportedQuery = "unsupported_query"
	CodeConflict         = "conflict"
	CodeReadOnly         = "read_only"

	// Refusals that are about the caller rather than the request.
	CodeUnauthorized    = "unauthorized"
	CodeTooManyAttempts = "too_many_attempts"
	CodeCrossOrigin     = "cross_origin"

	// Refusals of a change to the account itself: the password confirming it
	// was wrong, the account is set by the environment rather than stored,
	// or the server is one a password has to stay on.
	CodeWrongPassword    = "wrong_password"
	CodeAccountFixed     = "account_fixed"
	CodePasswordRequired = "password_required"

	// Refusals that are about the server: it has not been set up yet, or it
	// has and the caller is trying to set it up again.
	CodeSetupRequired = "setup_required"
	CodeAlreadySetUp  = "already_set_up"

	// A publish can be refused outright, need a warning acknowledged, or
	// fail partway. They are told apart because the client acts on each
	// differently.
	CodePublishRefused           = "publish_refused"
	CodePublishNeedsConfirmation = "publish_needs_confirmation"
	CodePublishFailed            = "publish_failed"
	CodeInternal                 = "internal"

	// CodeBuildFailed reports a site that could not be built as it stands,
	// such as one with content the index refused.
	CodeBuildFailed = "build_failed"
)

// errNothingWritten reports a store that accepted a write and reported no
// file, which would leave a client with no link to insert.
var errNothingWritten = errors.New("api: the store wrote nothing")

// ErrorBody is what every failing response carries.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes one failure.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Field names the query parameter or body field at fault, when one is.
	Field string `json:"field,omitempty"`
}

// fail writes an error response.
func fail(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// failField is [fail] for an error a client can attribute to one input.
func failField(w http.ResponseWriter, status int, code, field, message string) {
	writeJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message, Field: field}})
}

// failErr maps a domain error to a response.
//
// An unsupported query is a client error worth 400 rather than a 500: the
// reader refuses queries it cannot answer instead of loading everything and
// filtering in Go, so this is a contract being enforced, not a malfunction.
func (s *Server) failErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrNotFound):
		fail(w, http.StatusNotFound, CodeNotFound, err.Error())
	case errors.Is(err, content.ErrUnsupportedQuery):
		fail(w, http.StatusBadRequest, CodeUnsupportedQuery, err.Error())
	default:
		// The detail goes to the log, not to the client: it may name paths on
		// the machine the server runs on.
		s.log.Error("request failed", "err", err)
		fail(w, http.StatusInternalServerError, CodeInternal, "internal error")
	}
}

// failQuery reports request parameters that could not be understood.
func (s *Server) failQuery(w http.ResponseWriter, err error) {
	if bad, ok := errors.AsType[*badParam](err); ok {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, bad.Field, bad.Message)
		return
	}
	s.failErr(w, err)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal","message":"encode failed"}}`, http.StatusInternalServerError)
		return
	}
	data = append(data, '\n')

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	// Admin data is never cacheable: the files underneath it change while the
	// browser holds the page open.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// discardLogger is used when a caller supplies none.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
