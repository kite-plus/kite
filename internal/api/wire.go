package api

import (
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/url"
	"github.com/kite-plus/kite/internal/schema"
)

// The types in this file are the published shape of the API, deliberately
// separate from the domain model they are projected from.
//
// Serializing content.Content directly would make every field of the domain
// model a promise to every client: renaming one would break them, and adding
// an API field would mean adding it to the domain. The projection costs a
// little code and buys the freedom to change the inside without changing the
// outside, which is what "v1 only ever gains fields" requires.

// Summary is one row of a listing.
type Summary struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Locale string `json:"locale,omitempty"`

	Excerpt    string              `json:"excerpt,omitempty"`
	Taxonomies map[string][]string `json:"taxonomies,omitempty"`

	// URL is where the item appears on the site; Locator is where its bytes
	// live. They are reported separately because they are genuinely
	// independent: moving a file does not move a page.
	URL     string `json:"url"`
	Locator string `json:"locator,omitempty"`

	Revision    string    `json:"revision"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt time.Time `json:"published_at,omitzero"`
}

// Item is one content item in full.
type Item struct {
	Summary

	Body       string         `json:"body"`
	BodyFormat string         `json:"body_format"`
	Meta       map[string]any `json:"meta,omitempty"`
	Aliases    []string       `json:"aliases,omitempty"`
	DeletedAt  time.Time      `json:"deleted_at,omitzero"`
}

// List is a page of results.
//
// There is no offset and no page number. A cursor is the only position this
// API will ever hand out, because over a set that is being edited an offset
// both repeats and skips rows, and a paging scheme cannot be withdrawn once
// clients depend on it.
type List[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`

	// Total is present only when the request asked for it, since counting is
	// a second query over the whole filtered set.
	Total *int `json:"total,omitempty"`
}

// TermCount is one row of a taxonomy aggregation.
type TermCount struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
	URL   string `json:"url"`
}

// ContentType describes one kind of content, including the field schema an
// admin form is generated from.
type ContentType struct {
	Kind       string        `json:"kind"`
	Label      string        `json:"label"`
	Dir        string        `json:"dir"`
	Route      string        `json:"route"`
	Layout     string        `json:"layout"`
	Taxonomies []string      `json:"taxonomies,omitempty"`
	Sortable   []string      `json:"sortable,omitempty"`
	Fields     schema.Schema `json:"fields,omitempty"`
}

// SiteInfo describes the project the server has open.
type SiteInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	BaseURL     string `json:"base_url"`
	Language    string `json:"language,omitempty"`

	// Store and Runtime name the cell of the store x runtime matrix this
	// server is running in, so a client can tell what it is talking to
	// without guessing from behavior.
	Store   string `json:"store"`
	Runtime string `json:"runtime"`

	Theme   string `json:"theme,omitempty"`
	Version string `json:"version,omitempty"`

	// Problems lists content the index refused to load, such as a file with
	// no id. The admin shows them rather than leaving a file silently absent.
	Problems []string `json:"problems,omitempty"`

	// Counts is the number of items per kind.
	Counts map[string]int `json:"counts,omitempty"`
}

// Taxonomy names one taxonomy and how many distinct terms it holds.
type Taxonomy struct {
	Name  string `json:"name"`
	Terms int    `json:"terms"`
	URL   string `json:"url"`
}

// summaryOf projects a domain summary onto the wire.
func summaryOf(s content.Summary, r *url.Resolver) Summary {
	out := Summary{
		ID:         string(s.ID),
		Kind:       string(s.Kind),
		Slug:       s.Slug,
		Title:      s.Title,
		Status:     string(s.Status),
		Locale:     s.Locale,
		Excerpt:    s.Excerpt,
		Taxonomies: s.Taxonomies,
		URL:        r.ForSummary(s),
		Locator:    string(s.Locator),
		Revision:   string(s.Revision),
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
	if s.PublishedAt != nil {
		out.PublishedAt = *s.PublishedAt
	}
	return out
}

// itemOf projects a full domain item onto the wire.
func itemOf(c *content.Content, r *url.Resolver) Item {
	out := Item{
		Summary:    summaryOf(c.Summarize(), r),
		Body:       c.Body.Raw,
		BodyFormat: string(c.Body.Format),
		Meta:       c.Meta,
		Aliases:    c.Aliases,
	}
	if c.DeletedAt != nil {
		out.DeletedAt = *c.DeletedAt
	}
	return out
}

func contentTypeOf(t *content.Type) ContentType {
	return ContentType{
		Kind:       string(t.Kind),
		Label:      t.Label,
		Dir:        t.Dir,
		Route:      t.Route,
		Layout:     string(t.Layout),
		Taxonomies: t.Taxonomies,
		Sortable:   t.Sortable,
		Fields:     t.Fields,
	}
}
