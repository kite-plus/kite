// Package content holds Kite's domain model. It must not import any storage,
// rendering, runtime or publisher package: everything here is shared by every
// store backend and every runtime.
package content

import (
	"fmt"
	"slices"
	"time"
)

// ID identifies a content item for its whole lifetime. It is independent of the
// item's path, slug and URL, all of which may change freely. See
// docs/design/architecture.md D1.
type ID string

// Kind names a content type, such as "post" or "page".
type Kind string

// Status is the author's intent for an item. It does not say whether the item
// is actually live: that is the publisher's DeliveryState. See D4.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

var knownStatuses = []Status{StatusDraft, StatusScheduled, StatusPublished, StatusArchived}

// Valid reports whether s is a recognized status.
func (s Status) Valid() bool { return slices.Contains(knownStatuses, s) }

// Format is the source format of a [Body].
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
)

// Body is the unrendered source of an item. Rendered output is always derived
// and never stored here.
type Body struct {
	Format Format
	Raw    string
}

// Revision is an opaque token used for optimistic concurrency. The file store
// uses a content hash, a SQL store uses a version counter. Callers must treat
// it as opaque and only ever compare it for equality.
type Revision string

// Locator says where an item's bytes live. For the file store it is the bundle
// directory or single file, relative to the project root, always slash
// separated. It is meaningless for SQL stores.
type Locator string

// Content is a single item of content of any kind.
type Content struct {
	ID     ID
	Kind   Kind
	Slug   string
	Title  string
	Status Status
	Body   Body

	// Meta holds the type-specific fields declared by ContentType.Fields.
	Meta map[string]any

	// Taxonomies maps a taxonomy name such as "tag" to plain term strings.
	// Terms are strings, not entities: see D3.
	Taxonomies map[string][]string

	// Aliases are former URLs that should redirect to this item.
	Aliases []string

	Locale string

	Locator  Locator
	Revision Revision

	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
	DeletedAt   *time.Time
}

// Summary is the projection returned by list queries. It deliberately omits
// Body so that listing thousands of items never loads their full text.
type Summary struct {
	ID          ID
	Kind        Kind
	Slug        string
	Title       string
	Status      Status
	Taxonomies  map[string][]string
	Locale      string
	Locator     Locator
	Revision    Revision
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
	Excerpt     string

	// Pinned is the one field a listing reads from an item's metadata: an
	// author who pins a post expects to see which one it is in a list.
	Pinned bool
}

// Summarize projects a full item into a [Summary].
func (c *Content) Summarize() Summary {
	return Summary{
		ID:          c.ID,
		Kind:        c.Kind,
		Slug:        c.Slug,
		Title:       c.Title,
		Status:      c.Status,
		Taxonomies:  cloneTaxonomies(c.Taxonomies),
		Locale:      c.Locale,
		Locator:     c.Locator,
		Revision:    c.Revision,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		PublishedAt: c.PublishedAt,
		Pinned:      c.Meta["pinned"] == true,
	}
}

// Terms returns the terms of one taxonomy, or nil.
func (c *Content) Terms(taxonomy string) []string { return c.Taxonomies[taxonomy] }

// IsPublic reports whether the author intends the item to be visible. Whether
// it actually is visible depends on the runtime and the publisher.
func (c *Content) IsPublic(now time.Time) bool {
	if c.DeletedAt != nil {
		return false
	}
	switch c.Status {
	case StatusPublished:
		return true
	case StatusScheduled:
		return c.PublishedAt != nil && !c.PublishedAt.After(now)
	default:
		return false
	}
}

// Validate checks the invariants every store must uphold.
func (c *Content) Validate() error {
	switch {
	case c.ID == "":
		return fmt.Errorf("%w: id is required", ErrInvalid)
	case !ValidID(string(c.ID)):
		return fmt.Errorf("%w: id %q is not a valid ULID", ErrInvalid, c.ID)
	case c.Kind == "":
		return fmt.Errorf("%w: %s: kind is required", ErrInvalid, c.ID)
	case c.Slug == "":
		return fmt.Errorf("%w: %s: slug is required", ErrInvalid, c.ID)
	case !c.Status.Valid():
		return fmt.Errorf("%w: %s: unknown status %q", ErrInvalid, c.ID, c.Status)
	case c.Status == StatusScheduled && c.PublishedAt == nil:
		return fmt.Errorf("%w: %s: scheduled status requires published_at", ErrInvalid, c.ID)
	}
	return nil
}

func cloneTaxonomies(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = slices.Clone(v)
	}
	return out
}
