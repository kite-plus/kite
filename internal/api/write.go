package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/kite-plus/kite/internal/content"
)

// maxBody bounds a request body. A post is text; anything this large is a
// mistake or an attack, and either way it should not be buffered.
const maxBody = 8 << 20

// handleCreate writes a new item.
func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	draft, ok := s.decodeDraft(w, r)
	if !ok {
		return
	}
	if draft.Kind == "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "kind", "kind is required")
		return
	}
	if view.Types.Get(content.Kind(draft.Kind)) == nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "kind", "unknown kind: "+draft.Kind)
		return
	}
	// A create must not carry If-Match: there is nothing yet to match, and
	// honoring it would mean inventing a version the client never saw.
	if r.Header.Get("If-Match") != "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "If-Match",
			"a create has nothing to match; omit the header")
		return
	}

	item := draft.contentOf("")
	res, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutContent{Content: item}},
		Message: "new: " + draft.Title,
	})
	if err != nil {
		s.failWrite(w, view, r, err)
		return
	}

	stored, err := view.Reader.Get(r.Context(), res.IDs[0])
	if err != nil {
		s.failErr(w, err)
		return
	}
	out := itemOf(stored, view.Resolver)
	w.Header().Set("ETag", etag(stored.Revision))
	w.Header().Set("Location", Prefix+"/contents/"+string(stored.ID))
	writeJSON(w, http.StatusCreated, out)
}

// handleUpdate replaces an item, refusing an edit made against a version that
// has since been replaced.
func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	id := content.ID(r.PathValue("id"))
	revision, ok := s.requireIfMatch(w, r)
	if !ok {
		return
	}
	draft, ok := s.decodeDraft(w, r)
	if !ok {
		return
	}

	// The kind is a property of where the bytes live, not something an edit
	// may change: moving a post to another type is a move, not an update.
	existing, err := view.Reader.Get(r.Context(), id)
	if err != nil {
		s.failErr(w, err)
		return
	}
	if draft.Kind != "" && draft.Kind != string(existing.Kind) {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "kind",
			"an update cannot change the kind")
		return
	}

	item := draft.contentOf(id)
	item.Kind = existing.Kind
	item.Locator = existing.Locator
	item.CreatedAt = existing.CreatedAt

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutContent{Content: item, IfRevision: revision}},
		Message: "edit: " + draft.Title,
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}

	stored, err := view.Reader.Get(r.Context(), id)
	if err != nil {
		s.failErr(w, err)
		return
	}
	w.Header().Set("ETag", etag(stored.Revision))
	writeJSON(w, http.StatusOK, itemOf(stored, view.Resolver))
}

// handleDelete removes an item and everything its bundle owns.
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	id := content.ID(r.PathValue("id"))
	revision, ok := s.requireIfMatch(w, r)
	if !ok {
		return
	}

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.DeleteContent{ID: id, IfRevision: revision}},
		Message: "delete: " + string(id),
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// apply runs a change set and brings the read model back into agreement with
// the files before the caller looks at it again.
func (s *Server) apply(r *http.Request, view View, cs content.ChangeSet) (content.Result, error) {
	res, err := view.Writer.Apply(r.Context(), cs)
	if err != nil {
		return res, err
	}
	if view.Refresh != nil {
		if err := view.Refresh(r.Context()); err != nil {
			// The bytes are already on disk, so this is not a failed write:
			// it is a stale reader, and saying the write failed would invite
			// the client to repeat it.
			s.log.Error("refresh after write failed", "err", err)
		}
	}
	return res, nil
}

// writable reports whether this deployment accepts writes, answering if not.
func (s *Server) writable(w http.ResponseWriter, view View) bool {
	if view.Writer != nil {
		return true
	}
	fail(w, http.StatusMethodNotAllowed, CodeReadOnly, "this server is read only")
	return false
}

// requireIfMatch reads the revision an edit was made against.
//
// The header is mandatory rather than optional-with-a-default, because the
// default would have to be "overwrite whatever is there", and a lost edit is
// not something a client should be able to cause by forgetting a header.
func (s *Server) requireIfMatch(w http.ResponseWriter, r *http.Request) (content.Revision, bool) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" {
		failField(w, http.StatusPreconditionRequired, CodeInvalidRequest, "If-Match",
			"send the revision this edit was made against")
		return "", false
	}
	if raw == "*" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "If-Match",
			"* would overwrite whatever is there; send a revision")
		return "", false
	}
	return content.Revision(strings.Trim(raw, `"`)), true
}

func (s *Server) decodeDraft(w http.ResponseWriter, r *http.Request) (Draft, bool) {
	return decodeJSON[Draft](s, w, r)
}

// decodeJSON reads exactly one JSON document from a request body.
func decodeJSON[T any](_ *Server, w http.ResponseWriter, r *http.Request) (T, bool) {
	var out T

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	// An unknown field is refused for the same reason an unknown query
	// parameter is: silently dropping it lets a client believe a value it
	// sent is being honored.
	dec.DisallowUnknownFields()

	if err := dec.Decode(&out); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "the body is too large")
			return out, false
		}
		fail(w, http.StatusBadRequest, CodeInvalidRequest, "malformed body: "+err.Error())
		return out, false
	}
	if err := dec.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, "the body carries more than one document")
		return out, false
	}
	return out, true
}

// failWrite answers a write that the store refused.
func (s *Server) failWrite(w http.ResponseWriter, view View, r *http.Request, err error) {
	if conflict, ok := errors.AsType[*content.ConflictError](err); ok {
		s.failConflict(w, view, r, conflict)
		return
	}
	if errors.Is(err, content.ErrNotFound) {
		fail(w, http.StatusNotFound, CodeNotFound, err.Error())
		return
	}
	// A store refusing the content itself is telling the client its input was
	// wrong, not that the server broke.
	if errors.Is(err, content.ErrInvalid) {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
		return
	}
	s.failErr(w, err)
}

// failConflict answers with what is on disk now, so the client can show the
// two versions side by side against the base it still holds.
func (s *Server) failConflict(w http.ResponseWriter, view View, r *http.Request, e *content.ConflictError) {
	body := ConflictBody{
		Error: ErrorDetail{
			Code:    CodeConflict,
			Message: "this item changed since it was loaded",
		},
		Conflict: Conflict{
			ExpectedRevision: string(e.Expected),
			ActualRevision:   string(e.Actual),
		},
	}
	if current, err := view.Reader.Get(r.Context(), e.ID); err == nil {
		theirs := itemOf(current, view.Resolver)
		body.Conflict.Theirs = &theirs
	}
	w.Header().Set("ETag", etag(e.Actual))
	writeJSON(w, http.StatusConflict, body)
}

// etag formats a revision as a strong entity tag.
func etag(rev content.Revision) string { return `"` + string(rev) + `"` }
