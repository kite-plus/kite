package api

import (
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/publish"
	"github.com/kite-plus/kite/internal/render/theme"
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

	Revision string `json:"revision"`
	// A time the item does not record is left out rather than sent as the
	// year 1.
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
	PublishedAt time.Time `json:"published_at,omitzero"`
	// ModifiedAt is when the source file last changed, which a list can show
	// for an item that records no time of its own. A single item leaves it
	// out.
	ModifiedAt time.Time `json:"modified_at,omitzero"`

	// Pinned marks an item its author pinned, for a list to show as such.
	Pinned bool `json:"pinned,omitempty"`
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
	// Layouts are the templates the active theme offers items of this type,
	// chosen by name in an item's meta as layout. Layout above is unrelated:
	// it says how the items are stored.
	Layouts []LayoutOption `json:"layouts,omitempty"`
}

// LayoutOption is one template an item may choose.
type LayoutOption struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// layoutsFor lists the layouts a theme offers items of one kind.
func layoutsFor(layouts []theme.Layout, kind string) []LayoutOption {
	var out []LayoutOption
	for _, l := range layouts {
		if !l.ForType(kind) {
			continue
		}
		label := l.Label
		if label == "" {
			label = l.Name
		}
		out = append(out, LayoutOption{Name: l.Name, Label: label, Description: l.Description})
	}
	return out
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

// TermDetail is one term with every item that carries it: what renaming or
// removing the term would change.
type TermDetail struct {
	Term string `json:"term"`
	// Count is how many items outside the trash carry the term, as the term
	// list counts it.
	Count int    `json:"count"`
	URL   string `json:"url"`
	// Items lists every item carrying the term, trashed ones included and
	// marked: a change to the term reaches them too, so that restoring one
	// does not bring back a term that was renamed or removed.
	Items []TermItem `json:"items"`
}

// TermItem is one item that carries a term.
type TermItem struct {
	Summary
	Trashed bool `json:"trashed,omitempty"`
}

// TermRename is the request that renames a term.
type TermRename struct {
	// Name is the term's new name. Naming a term that exists merges the two.
	Name string `json:"name"`
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
		ModifiedAt: s.ModifiedAt,
		Pinned:     s.Pinned,
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

// Draft is the editable part of an item: exactly what a client may send.
//
// Every field the server owns is absent on purpose. The id comes from the
// path or is generated, the revision travels in If-Match, the timestamps
// belong to the store, and the locator is a physical address that only an
// explicit move changes. Accepting them would mean quietly ignoring them,
// which is worse than refusing them: a client would go on sending a value it
// believes is being honored.
type Draft struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Slug   string `json:"slug,omitempty"`
	Status string `json:"status"`
	Body   string `json:"body"`

	Meta       map[string]any      `json:"meta,omitempty"`
	Taxonomies map[string][]string `json:"taxonomies,omitempty"`
	Aliases    []string            `json:"aliases,omitempty"`
	Locale     string              `json:"locale,omitempty"`

	PublishedAt time.Time `json:"published_at,omitzero"`
}

// ConflictBody is the 409 response.
//
// It carries what is on disk now. The client is not sent a base version
// because the file store has no history to read one from: what the client
// loaded is the base, and it still has it. Pretending otherwise would mean
// caching every version the server ever handed out.
type ConflictBody struct {
	Error    ErrorDetail `json:"error"`
	Conflict Conflict    `json:"conflict"`
}

// Conflict describes an edit that was made against a version that has since
// been replaced.
type Conflict struct {
	ExpectedRevision string `json:"expected_revision"`
	ActualRevision   string `json:"actual_revision"`

	// Theirs is the item as it now stands on disk.
	Theirs *Item `json:"theirs,omitempty"`
}

// contentOf turns a draft into a domain item.
func (d Draft) contentOf(id content.ID) *content.Content {
	item := &content.Content{
		ID:         id,
		Kind:       content.Kind(d.Kind),
		Slug:       d.Slug,
		Title:      d.Title,
		Status:     content.Status(d.Status),
		Body:       content.Body{Format: content.FormatMarkdown, Raw: d.Body},
		Meta:       d.Meta,
		Taxonomies: d.Taxonomies,
		Aliases:    d.Aliases,
		Locale:     d.Locale,
	}
	if !d.PublishedAt.IsZero() {
		published := d.PublishedAt
		item.PublishedAt = &published
	}
	return item
}

// Media is a file stored beside a page.
type Media struct {
	Name string `json:"name"`

	// Path is where the bytes live in the repository; URL is where the file
	// appears on the site; Link is what belongs in the markdown.
	Path string `json:"path"`
	URL  string `json:"url"`
	Link string `json:"link"`

	Size int    `json:"size"`
	Type string `json:"type,omitempty"`
}

// Settings is what a project exposes to a configuration form.
type Settings struct {
	Site  SiteSettings  `json:"site"`
	Theme ThemeSettings `json:"theme"`

	// Writable lists the paths this API accepts, so a client can tell what it
	// may offer rather than discovering it by being refused.
	Writable []string `json:"writable"`
}

// SiteSettings is the site's own description.
type SiteSettings struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	BaseURL     string `json:"base_url"`
	Language    string `json:"language,omitempty"`
}

// ThemeSettings carries a theme's declared settings and their current values.
type ThemeSettings struct {
	Name   string         `json:"name"`
	Schema schema.Schema  `json:"schema,omitempty"`
	Values map[string]any `json:"values,omitempty"`
}

// PublishBody names what to publish.
type PublishBody struct {
	// IDs name items; Paths name files directly, for anything that is not an
	// item, such as the configuration.
	IDs   []string `json:"ids,omitempty"`
	Paths []string `json:"paths,omitempty"`

	Message string `json:"message,omitempty"`

	// Push sends the commit onward. Leaving the machine is a decision of its
	// own, so it is asked for rather than assumed.
	Push bool `json:"push"`

	// Force proceeds despite warnings that have already been shown.
	Force bool `json:"force,omitempty"`

	// SkipHooks commits without running the repository's commit hooks, for
	// an author who has read why one refused and decided to publish anyway.
	SkipHooks bool `json:"skip_hooks,omitempty"`
}

// PushBody asks for what is already committed to be pushed.
type PushBody struct {
	// Rebase replays the one unpushed commit on top of a remote that has
	// moved on, when the remote changed nothing that commit changed. It is
	// asked for, never assumed, because it rewrites that commit.
	Rebase bool `json:"rebase,omitempty"`
}

// PublishRefused is returned when a publish did not fully happen.
//
// It carries the plan, so a client can show every reason at once, and what
// did happen, because a push can fail after its commit succeeded and an
// author should not be invited to repeat a commit they already have.
type PublishRefused struct {
	Error ErrorDetail     `json:"error"`
	Plan  *publish.Plan   `json:"plan,omitempty"`
	Done  *publish.Result `json:"done,omitempty"`

	// Problem is the publisher's own account of what stopped it, with a
	// code a client can explain and what to do about it.
	Problem *publish.Problem `json:"problem,omitempty"`
}
