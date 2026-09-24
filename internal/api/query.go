package api

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/content"
)

// badParam names the query parameter a request got wrong.
type badParam struct {
	Field   string
	Message string
}

func (e *badParam) Error() string { return e.Field + ": " + e.Message }

// knownParams is every parameter the listing endpoint accepts.
var knownParams = []string{
	"kind", "status", "locale", "id",
	"term", "term_all",
	"published_from", "published_to", "updated_from", "updated_to",
	"q", "include_deleted", "deleted_only", "sort", "cursor", "limit", "count",
}

// parseQuery turns request parameters into a content query.
//
// A misspelled parameter is refused rather than ignored, because ignoring one
// returns a page that looks right and is not: ?stat=draft would quietly list
// everything, and the client would have no way to tell.
func parseQuery(v url.Values) (content.Query, error) {
	var q content.Query

	for name := range v {
		if !slices.Contains(knownParams, name) {
			return q, &badParam{Field: name, Message: "unknown parameter"}
		}
	}

	for _, k := range v["kind"] {
		q.Kinds = append(q.Kinds, content.Kind(k))
	}
	for _, s := range v["status"] {
		st := content.Status(s)
		if !st.Valid() {
			return q, &badParam{Field: "status", Message: fmt.Sprintf("unknown status %q", s)}
		}
		q.Statuses = append(q.Statuses, st)
	}
	q.Locales = append(q.Locales, v["locale"]...)
	for _, id := range v["id"] {
		q.IDs = append(q.IDs, content.ID(id))
	}

	var err error
	if q.TermsAny, err = parseTerms(v["term"], "term"); err != nil {
		return q, err
	}
	if q.TermsAll, err = parseTerms(v["term_all"], "term_all"); err != nil {
		return q, err
	}

	if q.Published, err = parseRange(v, "published"); err != nil {
		return q, err
	}
	if q.Updated, err = parseRange(v, "updated"); err != nil {
		return q, err
	}

	q.Text = strings.TrimSpace(v.Get("q"))

	if raw := v.Get("include_deleted"); raw != "" {
		if q.IncludeDeleted, err = strconv.ParseBool(raw); err != nil {
			return q, &badParam{Field: "include_deleted", Message: "expected true or false"}
		}
	}
	if raw := v.Get("deleted_only"); raw != "" {
		if q.DeletedOnly, err = strconv.ParseBool(raw); err != nil {
			return q, &badParam{Field: "deleted_only", Message: "expected true or false"}
		}
	}

	if q.Sort, err = parseSort(v.Get("sort")); err != nil {
		return q, err
	}

	q.Cursor = v.Get("cursor")
	if q.Cursor != "" {
		// Decoding here turns a mangled cursor into a named 400 rather than
		// an opaque failure from inside the reader.
		if _, err := content.DecodeCursor(q.Cursor); err != nil {
			return q, &badParam{Field: "cursor", Message: "malformed cursor"}
		}
	}

	if raw := v.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return q, &badParam{Field: "limit", Message: "expected a positive integer"}
		}
		q.Limit = n
	}
	return q, nil
}

// parseTerms reads repeated "taxonomy:term" parameters.
func parseTerms(values []string, field string) (map[string][]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string][]string, len(values))
	for _, raw := range values {
		taxonomy, term, ok := strings.Cut(raw, ":")
		if !ok || taxonomy == "" || term == "" {
			return nil, &badParam{Field: field, Message: fmt.Sprintf("expected taxonomy:term, got %q", raw)}
		}
		out[taxonomy] = append(out[taxonomy], term)
	}
	return out, nil
}

// parseRange reads <name>_from and <name>_to as RFC 3339 instants.
func parseRange(v url.Values, name string) (*content.Range, error) {
	from, err := parseTime(v, name+"_from")
	if err != nil {
		return nil, err
	}
	to, err := parseTime(v, name+"_to")
	if err != nil {
		return nil, err
	}
	if from == nil && to == nil {
		return nil, nil
	}
	return &content.Range{From: from, To: to}, nil
}

func parseTime(v url.Values, field string) (*time.Time, error) {
	raw := v.Get(field)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, &badParam{Field: field, Message: "expected an RFC 3339 timestamp"}
	}
	return &t, nil
}

// parseSort reads a comma separated ordering, where a leading "-" means
// descending: "-published_at,title".
//
// Unknown fields are left for the reader to reject, so that the list of
// sortable fields lives in one place.
func parseSort(raw string) ([]content.SortKey, error) {
	if raw == "" {
		return nil, nil
	}
	var keys []content.SortKey
	for part := range strings.SplitSeq(raw, ",") {
		field := strings.TrimSpace(part)
		if field == "" {
			return nil, &badParam{Field: "sort", Message: "empty sort key"}
		}
		field, desc := strings.CutPrefix(field, "-")
		keys = append(keys, content.SortKey{Field: field, Desc: desc})
	}
	return keys, nil
}
