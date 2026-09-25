package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/kite-plus/kite/internal/content"
)

// A term is a string on the items that carry it, not a record of its own
// (see D3), so renaming or removing one rewrites every item carrying it. The
// client first reads the term, which lists those items and carries an ETag
// fingerprinting them; the change sends that ETag back, and is refused if
// any of the items changed, or an item gained or lost the term, since.

// handleTerm reports one term and every item that carries it.
func (s *Server) handleTerm(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	taxonomy, term, ok := termPath(w, view, r)
	if !ok {
		return
	}
	s.writeTerm(w, r, view, taxonomy, term)
}

// handleRenameTerm renames a term on every item that carries it, merging it
// into another term when the new name is already in use.
func (s *Server) handleRenameTerm(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	taxonomy, term, ok := termPath(w, view, r)
	if !ok {
		return
	}
	revision, ok := s.requireIfMatch(w, r)
	if !ok {
		return
	}
	body, ok := decodeJSON[TermRename](s, w, r)
	if !ok {
		return
	}

	name := strings.TrimSpace(body.Name)
	switch {
	case name == "":
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "name", "a term needs a name")
		return
	case strings.ContainsFunc(name, unicode.IsControl):
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "name",
			"a term name cannot hold line breaks or other control characters")
		return
	case name == term:
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "name", "the term already has this name")
		return
	}

	if !s.changeTerm(w, r, view, taxonomy, term, name, revision, "rename "+taxonomy+": "+term+" -> "+name) {
		return
	}
	s.writeTerm(w, r, s.src(), taxonomy, name)
}

// handleRemoveTerm takes a term off every item that carries it. The items
// themselves stay.
func (s *Server) handleRemoveTerm(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	taxonomy, term, ok := termPath(w, view, r)
	if !ok {
		return
	}
	revision, ok := s.requireIfMatch(w, r)
	if !ok {
		return
	}

	if !s.changeTerm(w, r, view, taxonomy, term, "", revision, "remove "+taxonomy+": "+term) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// termPath reads the taxonomy and term a request names, refusing a taxonomy
// the project does not declare.
func termPath(w http.ResponseWriter, view View, r *http.Request) (string, string, bool) {
	taxonomy := r.PathValue("taxonomy")
	if !slices.Contains(view.Types.TaxonomyNames(), taxonomy) {
		fail(w, http.StatusNotFound, CodeNotFound, "no such taxonomy: "+taxonomy)
		return "", "", false
	}
	return taxonomy, r.PathValue("term"), true
}

// writeTerm answers with a term, its carriers and the ETag a change sends back.
func (s *Server) writeTerm(w http.ResponseWriter, r *http.Request, view View, taxonomy, term string) {
	items, trashed, err := carriers(r.Context(), view.Reader, taxonomy, term)
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(items) == 0 {
		fail(w, http.StatusNotFound, CodeNotFound, "no item carries the term: "+term)
		return
	}

	out := TermDetail{
		Term:  term,
		Count: len(items) - len(trashed),
		URL:   view.Resolver.ForTerm(taxonomy, term, view.Site.Language),
		Items: make([]TermItem, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, TermItem{
			Summary: summaryOf(item, view.Resolver),
			Trashed: trashed[item.ID],
		})
	}
	w.Header().Set("ETag", etag(termRevision(items)))
	writeJSON(w, http.StatusOK, out)
}

// changeTerm renames the term to to, or removes it when to is empty, on every
// item that carries it, in one change set, provided they are still the items
// the client was shown.
func (s *Server) changeTerm(w http.ResponseWriter, r *http.Request, view View,
	taxonomy, term, to string, revision content.Revision, message string,
) bool {
	items, _, err := carriers(r.Context(), view.Reader, taxonomy, term)
	if err != nil {
		s.failErr(w, err)
		return false
	}
	if len(items) == 0 {
		fail(w, http.StatusNotFound, CodeNotFound, "no item carries the term: "+term)
		return false
	}
	if current := termRevision(items); current != revision {
		w.Header().Set("ETag", etag(current))
		fail(w, http.StatusConflict, CodeConflict,
			"the items carrying this term changed since it was read")
		return false
	}

	cs := content.ChangeSet{Message: message}
	for _, item := range items {
		cs.Add(content.ChangeTerm{
			ID: item.ID, IfRevision: item.Revision, Taxonomy: taxonomy, Term: term, To: to,
		})
	}

	if _, err := s.apply(r, view, cs); err != nil {
		// One of the items changed between the check above and the write.
		// The store checks every revision before it writes anything, so
		// nothing was changed and the client can read the term again.
		if _, ok := errors.AsType[*content.ConflictError](err); ok {
			fail(w, http.StatusConflict, CodeConflict,
				"the items carrying this term changed since it was read")
			return false
		}
		s.failWrite(w, view, r, err)
		return false
	}
	return true
}

// carriers lists every item that carries a term, trashed ones included, and
// which of them are trashed.
func carriers(ctx context.Context, reader content.Reader, taxonomy, term string) ([]content.Summary, map[content.ID]bool, error) {
	carrying := map[string][]string{taxonomy: {term}}
	items, err := everyItem(ctx, reader, content.Query{TermsAll: carrying, IncludeDeleted: true})
	if err != nil {
		return nil, nil, err
	}
	deleted, err := everyItem(ctx, reader, content.Query{TermsAll: carrying, DeletedOnly: true})
	if err != nil {
		return nil, nil, err
	}
	trashed := make(map[content.ID]bool, len(deleted))
	for _, item := range deleted {
		trashed[item.ID] = true
	}
	return items, trashed, nil
}

// everyItem walks a query to its end rather than stopping at one page.
func everyItem(ctx context.Context, reader content.Reader, q content.Query) ([]content.Summary, error) {
	q.Limit = content.MaxLimit
	var out []content.Summary
	for {
		page, err := reader.Query(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if !page.HasMore {
			return out, nil
		}
		q.Cursor = page.NextCursor
	}
}

// termRevision fingerprints a term by the items carrying it and their
// revisions, so an edit to any of them, or an item gaining or losing the
// term, changes it.
func termRevision(items []content.Summary) content.Revision {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, string(item.ID)+" "+string(item.Revision))
	}
	slices.Sort(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return content.Revision("term:" + hex.EncodeToString(sum[:16]))
}
