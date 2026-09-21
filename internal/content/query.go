package content

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// ErrUnsupportedQuery is returned by a reader that cannot satisfy a query.
//
// A backend must return this rather than quietly loading everything and
// filtering in Go: a silent fallback is the one reliable way for O(n) scans to
// creep back in unnoticed.
var ErrUnsupportedQuery = errors.New("content: unsupported query")

// Sort fields accepted in a [SortKey].
const (
	SortPublishedAt = "published_at"
	SortUpdatedAt   = "updated_at"
	SortCreatedAt   = "created_at"
	SortTitle       = "title"
	SortSlug        = "slug"
	SortID          = "id"
)

var sortFields = []string{
	SortPublishedAt, SortUpdatedAt, SortCreatedAt, SortTitle, SortSlug, SortID,
}

// SortKey is one level of an ordering.
type SortKey struct {
	Field string
	Desc  bool
}

// DefaultSort orders newest first. ID breaks ties so that the ordering is
// total, which is what makes cursor pagination stable.
func DefaultSort() []SortKey {
	return []SortKey{
		{Field: SortPublishedAt, Desc: true},
		{Field: SortID, Desc: true},
	}
}

// Range is an inclusive time window. A nil bound is open.
type Range struct {
	From *time.Time
	To   *time.Time
}

// Query is a specification object rather than a method per query, so that the
// reader interface stays small as filters accumulate.
type Query struct {
	Kinds    []Kind
	Statuses []Status
	Locales  []string
	IDs      []ID

	// TermsAny matches items carrying at least one of the listed terms in the
	// given taxonomy; TermsAll requires every listed term.
	TermsAny map[string][]string
	TermsAll map[string][]string

	// Published restricts by publication time, Updated by last modification.
	Published *Range
	Updated   *Range

	// Text is a free text query over title, excerpt and body.
	Text string

	// IncludeDeleted returns soft deleted items as well.
	IncludeDeleted bool

	Sort   []SortKey
	Cursor string
	Limit  int
}

// DefaultLimit is used when a query does not set one.
const DefaultLimit = 50

// MaxLimit caps a single page.
const MaxLimit = 500

// Normalize fills in defaults and clamps the limit. It returns an error when
// the query asks for something structurally invalid.
func (q *Query) Normalize() error {
	if len(q.Sort) == 0 {
		q.Sort = DefaultSort()
	}
	for _, k := range q.Sort {
		if !slices.Contains(sortFields, k.Field) {
			return fmt.Errorf("%w: unknown sort field %q", ErrUnsupportedQuery, k.Field)
		}
	}
	// A total ordering is required for cursor pagination to be stable.
	if !slices.ContainsFunc(q.Sort, func(k SortKey) bool { return k.Field == SortID }) {
		q.Sort = append(q.Sort, SortKey{Field: SortID, Desc: q.Sort[len(q.Sort)-1].Desc})
	}
	switch {
	case q.Limit <= 0:
		q.Limit = DefaultLimit
	case q.Limit > MaxLimit:
		q.Limit = MaxLimit
	}
	for _, s := range q.Statuses {
		if !s.Valid() {
			return fmt.Errorf("%w: unknown status %q", ErrUnsupportedQuery, s)
		}
	}
	return nil
}

// Cursor is an opaque position in a result set. It carries one value per sort
// key, so paging stays correct even while items are being edited.
//
// Offset pagination is deliberately not offered: over a mutable set it both
// repeats and skips rows, and once a public API exposes it the behavior
// cannot be fixed.
type Cursor struct {
	Values []string `json:"v"`
}

// Encode renders the cursor as a URL-safe opaque string.
func (c Cursor) Encode() string {
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor parses a cursor produced by [Cursor.Encode].
func DecodeCursor(s string) (Cursor, error) {
	var c Cursor
	if s == "" {
		return c, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, fmt.Errorf("content: malformed cursor: %w", err)
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("content: malformed cursor: %w", err)
	}
	return c, nil
}

// Page is one page of results plus the cursor that continues it.
type Page[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}

// Capabilities describes what a reader can do, so callers can degrade
// deliberately instead of discovering limits through errors.
type Capabilities struct {
	FullText  bool
	Aggregate bool
}

// TermCount is one row of a taxonomy aggregation.
type TermCount struct {
	Taxonomy string
	Term     string
	Count    int
}

// Reader is the read side of the content store.
//
// There is exactly one implementation. In file mode an indexer derives its rows
// from markdown files; in SQL mode the writer populates them directly. Sharing
// one read model is what keeps static and dynamic rendering from diverging.
type Reader interface {
	Get(ctx context.Context, id ID) (*Content, error)
	GetBySlug(ctx context.Context, kind Kind, slug string) (*Content, error)
	Query(ctx context.Context, q Query) (Page[Summary], error)

	// Count reports the size of the set a query selects, disregarding its
	// cursor and limit. It is a separate call because a caller that only
	// wants the next page should not pay to count the rest.
	Count(ctx context.Context, q Query) (int, error)

	CountTerms(ctx context.Context, taxonomy string, q Query) ([]TermCount, error)
	Caps() Capabilities
}
