// Package schema defines the declarative field schema shared by content types,
// theme settings and plugin settings. A single schema definition drives both
// validation and automatic form generation in the admin UI.
package schema

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Type identifies the kind of value a [Field] holds and the control the admin
// renders for it.
type Type string

const (
	TypeString      Type = "string"
	TypeText        Type = "text"
	TypeNumber      Type = "number"
	TypeBoolean     Type = "boolean"
	TypeColor       Type = "color"
	TypeSelect      Type = "select"
	TypeMultiSelect Type = "multiselect"
	TypeImage       Type = "image"
	TypeURL         Type = "url"
	TypeDate        Type = "date"
	TypeCode        Type = "code"
	TypeGroup       Type = "group"
	TypeRepeat      Type = "repeat"

	// TypeSection gathers fields under a heading in the admin. It holds no
	// value: the fields in it are stored beside it, as if it were not there,
	// so a theme can arrange its form without changing the keys it reads.
	TypeSection Type = "section"
)

// Option is a choice offered by [TypeSelect] and [TypeMultiSelect] fields.
// On a [TypeColor] field the options are suggested colors, not a limit.
type Option struct {
	Value string `json:"value" yaml:"value"`
	Label string `json:"label" yaml:"label"`
}

// Field is one entry in a [Schema].
type Field struct {
	Key         string   `json:"key" yaml:"key"`
	Type        Type     `json:"type" yaml:"type"`
	Label       string   `json:"label,omitempty" yaml:"label,omitempty"`
	Help        string   `json:"help,omitempty" yaml:"help,omitempty"`
	Default     any      `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool     `json:"required,omitzero" yaml:"required,omitempty"`
	Placeholder string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Options     []Option `json:"options,omitempty" yaml:"options,omitempty"`

	Min  *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max  *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Step *float64 `json:"step,omitempty" yaml:"step,omitempty"`

	// Language selects the syntax highlighting mode for [TypeCode].
	Language string `json:"language,omitempty" yaml:"language,omitempty"`

	// Fields holds the nested schema of [TypeGroup], [TypeRepeat] and
	// [TypeSection].
	Fields Schema `json:"fields,omitempty" yaml:"fields,omitempty"`

	// ShowIf hides the field until every listed key holds the given value.
	ShowIf map[string]any `json:"showIf,omitempty" yaml:"showIf,omitempty"`
}

// Schema is an ordered list of fields. Order is significant: the admin renders
// fields in declaration order.
type Schema []Field

var nestingTypes = []Type{TypeGroup, TypeRepeat}

var knownTypes = []Type{
	TypeString, TypeText, TypeNumber, TypeBoolean, TypeColor,
	TypeSelect, TypeMultiSelect, TypeImage, TypeURL, TypeDate,
	TypeCode, TypeGroup, TypeRepeat, TypeSection,
}

// Validate reports whether the schema is well formed. It checks for unknown
// types, duplicate keys within a level, select fields without options and
// defaults their own field would refuse.
func (s Schema) Validate() error {
	return s.validate(make(map[string]bool), true)
}

// validate checks one level of a schema. The fields of a section are stored
// on the level the section sits on, so they share its keys.
func (s Schema) validate(seen map[string]bool, top bool) error {
	for i, f := range s {
		where := fmt.Sprintf("field %d (%q)", i, f.Key)
		if f.Key == "" {
			return fmt.Errorf("%s: key is required", where)
		}
		if seen[f.Key] {
			return fmt.Errorf("%s: duplicate key", where)
		}
		seen[f.Key] = true

		if !slices.Contains(knownTypes, f.Type) {
			return fmt.Errorf("%s: unknown type %q", where, f.Type)
		}
		if f.Type == TypeSection {
			if !top {
				return fmt.Errorf("%s: a section can only sit at the top of a schema", where)
			}
			if len(f.Fields) == 0 {
				return fmt.Errorf("%s: type %q requires nested fields", where, f.Type)
			}
			if f.Default != nil || f.Required {
				return fmt.Errorf("%s: a section holds no value, so it takes no default and cannot be required", where)
			}
			if err := f.Fields.validate(seen, false); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			continue
		}
		if (f.Type == TypeSelect || f.Type == TypeMultiSelect) && len(f.Options) == 0 {
			return fmt.Errorf("%s: type %q requires options", where, f.Type)
		}
		if slices.Contains(nestingTypes, f.Type) {
			if len(f.Fields) == 0 {
				return fmt.Errorf("%s: type %q requires nested fields", where, f.Type)
			}
			if err := f.Fields.validate(make(map[string]bool), false); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		} else if len(f.Fields) > 0 {
			return fmt.Errorf("%s: type %q cannot have nested fields", where, f.Type)
		}
		if f.Default != nil {
			if problem := f.check(f.Default); problem != "" {
				return fmt.Errorf("%s: default: %s", where, problem)
			}
		}
	}
	return nil
}

// values lists the fields that hold values on this level, with the fields of
// each section in its place.
func (s Schema) values() []*Field {
	var out []*Field
	for i := range s {
		if s[i].Type == TypeSection {
			out = append(out, s[i].Fields.values()...)
			continue
		}
		out = append(out, &s[i])
	}
	return out
}

// Field returns the field with the given key, or nil. The fields of a section
// are found as if the section were not there, and a section is not a field.
func (s Schema) Field(key string) *Field {
	for _, f := range s.values() {
		if f.Key == key {
			return f
		}
	}
	return nil
}

// Lookup returns the field a path of keys names, descending into groups, as
// in ["social", "github"]. The entries of a repeat are not addressed one by
// one, so a path into a repeat names nothing.
func (s Schema) Lookup(path []string) *Field {
	if len(path) == 0 {
		return nil
	}
	f := s.Field(path[0])
	switch {
	case f == nil || len(path) == 1:
		return f
	case f.Type == TypeGroup:
		return f.Fields.Lookup(path[1:])
	default:
		return nil
	}
}

// Defaults returns the default value of every field that declares one,
// applied recursively to groups. A repeat without a default is an empty list.
func (s Schema) Defaults() map[string]any {
	out := make(map[string]any)
	for _, f := range s.values() {
		switch f.Type {
		case TypeGroup:
			if nested := f.Fields.Defaults(); len(nested) > 0 {
				out[f.Key] = nested
			}
		case TypeRepeat:
			value, _ := f.resolve(nil, false)
			out[f.Key] = value
		default:
			if f.Default != nil {
				out[f.Key] = f.Default
			}
		}
	}
	return out
}

// Resolve returns the values a template reads. Each field holds a value of
// its declared type: the stored one when it can be read as that type, and the
// field's default otherwise. A group is always a map, so a template can reach
// into it without first checking it is there. Keys the schema does not
// declare are passed through untouched.
//
// It makes the best of whatever a file holds, since a configuration written by
// hand, or for another theme, is no reason for a page to fail to render.
func (s Schema) Resolve(stored map[string]any) map[string]any {
	out := make(map[string]any, len(stored))
	for key, value := range stored {
		out[key] = value
	}
	for _, f := range s.values() {
		value, present := stored[f.Key]
		if resolved, ok := f.resolve(value, present); ok {
			out[f.Key] = resolved
		} else {
			delete(out, f.Key)
		}
	}
	return out
}

// resolve reads a stored value as the field's type, falling back to its
// default. It reports false when the field has neither.
func (f *Field) resolve(value any, present bool) (any, bool) {
	if present && value != nil {
		if read, ok := f.read(value); ok {
			return read, true
		}
	}
	switch f.Type {
	case TypeGroup:
		return f.Fields.Resolve(nil), true
	case TypeRepeat:
		if f.Default != nil {
			if read, ok := f.read(f.Default); ok {
				return read, true
			}
		}
		return []any{}, true
	}
	if f.Default != nil {
		return f.Default, true
	}
	return nil, false
}

// read takes a value as the field's type, reporting false when it cannot be.
func (f *Field) read(value any) (any, bool) {
	switch f.Type {
	case TypeBoolean:
		switch v := value.(type) {
		case bool:
			return v, true
		case string:
			if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
				return b, true
			}
		}
		return nil, false

	case TypeNumber:
		switch v := value.(type) {
		case int, int64, uint64, float64:
			return v, true
		case string:
			if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return n, true
			}
		}
		return nil, false

	case TypeSelect:
		text, ok := scalar(value)
		if !ok || !f.offers(text) {
			return nil, false
		}
		return text, true

	case TypeMultiSelect:
		var items []any
		switch v := value.(type) {
		case []any:
			items = v
		case string:
			items = []any{v}
		default:
			return nil, false
		}
		out := make([]any, 0, len(items))
		for _, item := range items {
			if text, ok := scalar(item); ok && f.offers(text) && !slices.Contains(out, any(text)) {
				out = append(out, text)
			}
		}
		return out, true

	case TypeGroup:
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		return f.Fields.Resolve(m), true

	case TypeRepeat:
		switch v := value.(type) {
		case []any:
			out := make([]any, 0, len(v))
			for _, item := range v {
				if entry, ok := f.entry(item); ok {
					out = append(out, entry)
				}
			}
			return out, true
		case map[string]any:
			return []any{f.Fields.Resolve(v)}, true
		case string:
			out := []any{}
			for _, line := range strings.Split(v, "\n") {
				if entry, ok := f.entry(line); ok {
					out = append(out, entry)
				}
			}
			return out, true
		}
		return nil, false

	default:
		text, ok := scalar(value)
		if !ok {
			return nil, false
		}
		return text, true
	}
}

// entry reads one entry of a repeat. A line of text is an entry too, its
// parts separated by "|" and taken as the fields in order, so a list a theme
// used to ask for as text, one "About | /about/" per line, reads as the list
// it always described.
func (f *Field) entry(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return f.Fields.Resolve(v), true
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, false
		}
		fields := f.Fields.values()
		stored := make(map[string]any)
		for i, part := range strings.Split(v, "|") {
			if i < len(fields) {
				stored[fields[i].Key] = strings.TrimSpace(part)
			}
		}
		return f.Fields.Resolve(stored), true
	}
	return nil, false
}

// scalar reads a value as text. A number or a boolean written without quotes
// is still text to a field that asks for text.
func scalar(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool, int, int64, uint64, float64:
		return fmt.Sprint(v), true
	case time.Time:
		if v.Equal(v.Truncate(24*time.Hour)) && v.Location() == time.UTC {
			return v.Format(time.DateOnly), true
		}
		return v.Format(time.RFC3339), true
	}
	return "", false
}

func (f *Field) offers(value string) bool {
	return slices.ContainsFunc(f.Options, func(o Option) bool { return o.Value == value })
}

var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// Check reports why a value cannot be stored in the field, or "" when it can.
//
// It is stricter than [Schema.Resolve], which makes the best of whatever a
// file holds: a form or a script writing a setting learns what was wrong with
// it, rather than seeing the value quietly ignored when the page is drawn.
func (f *Field) Check(value any) string {
	if f.Required && blank(value) {
		return "a value is required"
	}
	if value == nil {
		return ""
	}
	return f.check(value)
}

func (f *Field) check(value any) string {
	switch f.Type {
	case TypeSection:
		return "a section holds no value"

	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return "want true or false"
		}

	case TypeNumber:
		n, ok := number(value)
		switch {
		case !ok:
			return "want a number"
		case f.Min != nil && n < *f.Min:
			return fmt.Sprintf("want at least %v", *f.Min)
		case f.Max != nil && n > *f.Max:
			return fmt.Sprintf("want at most %v", *f.Max)
		}

	case TypeSelect:
		if text, ok := value.(string); !ok || !f.offers(text) {
			return "want one of " + f.choices()
		}

	case TypeMultiSelect:
		items, ok := value.([]any)
		if !ok {
			return "want a list of " + f.choices()
		}
		for _, item := range items {
			if text, ok := item.(string); !ok || !f.offers(text) {
				return "want a list of " + f.choices()
			}
		}

	case TypeColor:
		if text, ok := value.(string); !ok || (text != "" && !hexColor.MatchString(text)) {
			return "want a color written as #rrggbb, such as #7d5c3c"
		}

	case TypeURL:
		text, ok := value.(string)
		if !ok {
			return "want a link"
		}
		return checkLink(text)

	case TypeDate:
		if text, ok := value.(string); !ok || (text != "" && !isDate(text)) {
			return "want a date such as 2026-09-01 or 2026-09-01T10:00"
		}

	case TypeGroup:
		m, ok := value.(map[string]any)
		if !ok {
			return "want a set of fields"
		}
		return f.Fields.check(m)

	case TypeRepeat:
		items, ok := value.([]any)
		if !ok {
			return "want a list"
		}
		for i, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				return fmt.Sprintf("entry %d: want a set of fields", i+1)
			}
			if problem := f.Fields.check(m); problem != "" {
				return fmt.Sprintf("entry %d: %s", i+1, problem)
			}
		}

	default:
		if _, ok := value.(string); !ok {
			return "want text"
		}
	}
	return ""
}

// check checks the values of a group or of one entry of a repeat.
func (s Schema) check(values map[string]any) string {
	for _, f := range s.values() {
		if problem := f.Check(values[f.Key]); problem != "" {
			return f.Key + ": " + problem
		}
	}
	return ""
}

func (f *Field) choices() string {
	values := make([]string, 0, len(f.Options))
	for _, o := range f.Options {
		values = append(values, strconv.Quote(o.Value))
	}
	return strings.Join(values, ", ")
}

func blank(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	}
	return false
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	}
	return 0, false
}

// checkLink accepts an address on the web, a mail or phone link and a path
// within the site. Anything else, a javascript: link above all, is refused
// before it reaches a page.
func checkLink(text string) string {
	if text == "" {
		return ""
	}
	u, err := url.Parse(text)
	if err != nil || strings.ContainsAny(text, " \t\r\n") {
		return "not a link"
	}
	switch strings.ToLower(u.Scheme) {
	case "":
		return ""
	case "http", "https":
		if u.Host == "" {
			return "a web address needs a host, such as https://example.com"
		}
		return ""
	case "mailto", "tel":
		return ""
	}
	return "want a link starting with https://, http://, mailto:, tel: or /"
}

func isDate(text string) bool {
	for _, layout := range []string{time.DateOnly, "2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339} {
		if _, err := time.Parse(layout, text); err == nil {
			return true
		}
	}
	return false
}
