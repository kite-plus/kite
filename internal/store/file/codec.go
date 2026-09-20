// Package file implements the content store backed by markdown files.
//
// The files are the source of truth: every write lands in the working tree
// where the user's editor and git see it immediately, and nothing here
// consults a database.
package file

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/frontmatter"
)

// Canonical front matter keys. Everything not reserved, and not the name of a
// taxonomy the type declares, is carried in Content.Meta.
const (
	keyID          = "id"
	keyTitle       = "title"
	keySlug        = "slug"
	keyStatus      = "status"
	keyCreatedAt   = "created_at"
	keyUpdatedAt   = "updated_at"
	keyPublishedAt = "published_at"
	keyDeletedAt   = "deleted_at"
	keyAliases     = "aliases"
	keyLocale      = "locale"
)

// Legacy keys accepted on read for compatibility with existing Hugo and Hexo
// content. They are never written back: a save migrates the file to the
// canonical spelling.
const (
	legacyDate    = "date"
	legacyLastmod = "lastmod"
	legacyDraft   = "draft"
)

var reservedKeys = []string{
	keyID, keyTitle, keySlug, keyStatus,
	keyCreatedAt, keyUpdatedAt, keyPublishedAt, keyDeletedAt,
	keyAliases, keyLocale,
	legacyDate, legacyLastmod, legacyDraft,
}

// writeOrder is the order canonical keys take when a file is created. Existing
// files keep whatever order they already have.
var writeOrder = []string{
	keyID, keyTitle, keySlug, keyStatus,
	keyCreatedAt, keyUpdatedAt, keyPublishedAt,
	keyAliases, keyLocale,
}

// Codec converts between markdown files and domain items.
type Codec struct {
	types *content.Registry
}

// NewCodec returns a codec bound to a type registry.
func NewCodec(types *content.Registry) *Codec { return &Codec{types: types} }

// Decode parses a source file into an item. Reading is deliberately lenient so
// that an existing Hugo or Hexo repository can be opened without migration;
// the only hard requirement is that the front matter is a mapping.
func (c *Codec) Decode(t *content.Type, loc content.Locator, data []byte) (*content.Content, error) {
	doc, err := frontmatter.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", loc, err)
	}

	item := &content.Content{
		Kind:     t.Kind,
		Locator:  loc,
		Revision: RevisionOf(data),
		Title:    doc.String(keyTitle),
		Slug:     doc.String(keySlug),
		Locale:   doc.String(keyLocale),
		Aliases:  doc.StringSlice(keyAliases),
		Body:     content.Body{Format: content.FormatMarkdown, Raw: doc.Body()},
	}

	if id := doc.String(keyID); id != "" {
		item.ID = content.ID(id)
	}
	if item.Slug == "" {
		item.Slug = defaultSlug(t, loc)
	}
	if item.Title == "" {
		item.Title = item.Slug
	}

	item.Status = decodeStatus(doc)
	item.CreatedAt = firstTime(doc, keyCreatedAt, legacyDate)
	item.UpdatedAt = firstTime(doc, keyUpdatedAt, legacyLastmod)
	if ts := firstTime(doc, keyPublishedAt, legacyDate); !ts.IsZero() {
		item.PublishedAt = &ts
	}
	if ts := firstTime(doc, keyDeletedAt); !ts.IsZero() {
		item.DeletedAt = &ts
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}

	item.Taxonomies = decodeTaxonomies(doc, t)
	item.Meta = decodeMeta(doc, t)
	return item, nil
}

// Encode renders an item back to file bytes.
//
// When existing holds the file's current bytes the update is surgical: keys
// whose values are unchanged are left byte for byte alone, so saving an item
// after editing its title produces a one line diff.
func (c *Codec) Encode(t *content.Type, item *content.Content, existing []byte) ([]byte, error) {
	doc, err := frontmatter.Parse(existing)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", item.Locator, err)
	}

	values := map[string]any{
		keyID:     string(item.ID),
		keyTitle:  item.Title,
		keySlug:   item.Slug,
		keyStatus: string(item.Status),
	}
	if ts, ok := timeValue(item.CreatedAt); ok {
		values[keyCreatedAt] = ts
	}
	if ts, ok := timeValue(item.UpdatedAt); ok {
		values[keyUpdatedAt] = ts
	}
	if item.PublishedAt != nil {
		if ts, ok := timeValue(*item.PublishedAt); ok {
			values[keyPublishedAt] = ts
		}
	}
	if len(item.Aliases) > 0 {
		values[keyAliases] = item.Aliases
	}
	if item.Locale != "" {
		values[keyLocale] = item.Locale
	}
	if err := doc.SetAll(writeOrder, values); err != nil {
		return nil, err
	}

	if item.PublishedAt == nil {
		doc.Delete(keyPublishedAt)
	}
	if item.DeletedAt != nil {
		ts, _ := timeValue(*item.DeletedAt)
		if err := doc.Set(keyDeletedAt, ts); err != nil {
			return nil, err
		}
	} else {
		doc.Delete(keyDeletedAt)
	}

	// Once a canonical status is written the legacy draft flag is redundant
	// and would contradict it on the next read.
	if _, ok := doc.Get(legacyDraft); ok {
		doc.Delete(legacyDraft)
	}

	for _, name := range t.Taxonomies {
		terms := item.Taxonomies[name]
		if len(terms) == 0 {
			doc.Delete(name)
			continue
		}
		if err := doc.Set(name, terms); err != nil {
			return nil, err
		}
	}

	for _, key := range slices.Sorted(maps.Keys(item.Meta)) {
		if err := doc.Set(key, item.Meta[key]); err != nil {
			return nil, err
		}
	}

	doc.SetBody(item.Body.Raw)
	return doc.Bytes()
}

func decodeStatus(doc *frontmatter.Document) content.Status {
	if s := content.Status(doc.String(keyStatus)); s.Valid() {
		return s
	}
	if v, ok := doc.Get(legacyDraft); ok {
		if draft, _ := v.(bool); draft {
			return content.StatusDraft
		}
	}
	return content.StatusPublished
}

func decodeTaxonomies(doc *frontmatter.Document, t *content.Type) map[string][]string {
	var out map[string][]string
	for _, name := range t.Taxonomies {
		terms := doc.StringSlice(name)
		if len(terms) == 0 {
			// A single scalar is a common shorthand for a one term list.
			if s := doc.String(name); s != "" {
				terms = []string{s}
			}
		}
		if len(terms) == 0 {
			continue
		}
		if out == nil {
			out = make(map[string][]string)
		}
		out[name] = terms
	}
	return out
}

func decodeMeta(doc *frontmatter.Document, t *content.Type) map[string]any {
	var out map[string]any
	for _, key := range doc.Keys() {
		if slices.Contains(reservedKeys, key) || slices.Contains(t.Taxonomies, key) {
			continue
		}
		v, ok := doc.Get(key)
		if !ok {
			continue
		}
		if out == nil {
			out = make(map[string]any)
		}
		out[key] = v
	}
	return out
}

// defaultSlug derives a slug from the file name when front matter omits one.
func defaultSlug(t *content.Type, loc content.Locator) string {
	name := string(loc)
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if t.Layout == content.LayoutSingleFile {
		name = strings.TrimSuffix(name, markdownExt)
	}
	return name
}

// timeFormats are tried in order. RFC3339 is canonical; the rest cover what
// existing static site generators emit.
var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func firstTime(doc *frontmatter.Document, keys ...string) time.Time {
	for _, k := range keys {
		v, ok := doc.Get(k)
		if !ok {
			continue
		}
		if ts, ok := parseTime(v); ok {
			return ts
		}
	}
	return time.Time{}
}

func parseTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		for _, layout := range timeFormats {
			if ts, err := time.Parse(layout, t); err == nil {
				return ts, true
			}
		}
	}
	return time.Time{}, false
}

// timeValue normalizes a timestamp for writing. Times are written as YAML
// timestamps rather than strings: writing a string where the file held a
// timestamp forces the encoder to quote it, which is exactly the kind of
// meaningless diff this package exists to avoid.
func timeValue(t time.Time) (time.Time, bool) {
	if t.IsZero() {
		return time.Time{}, false
	}
	return t.UTC(), true
}
