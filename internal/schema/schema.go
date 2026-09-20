// Package schema defines the declarative field schema shared by content types,
// theme settings and plugin settings. A single schema definition drives both
// validation and automatic form generation in the admin UI.
package schema

import (
	"fmt"
	"slices"
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
)

// Option is a choice offered by [TypeSelect] and [TypeMultiSelect] fields.
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

	Min *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max *float64 `json:"max,omitempty" yaml:"max,omitempty"`

	// Language selects the syntax highlighting mode for [TypeCode].
	Language string `json:"language,omitempty" yaml:"language,omitempty"`

	// Fields holds the nested schema of [TypeGroup] and [TypeRepeat].
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
	TypeCode, TypeGroup, TypeRepeat,
}

// Validate reports whether the schema is well formed. It checks for unknown
// types, duplicate keys within a level, and select fields without options.
func (s Schema) Validate() error {
	seen := make(map[string]struct{}, len(s))
	for i, f := range s {
		where := fmt.Sprintf("field %d (%q)", i, f.Key)
		if f.Key == "" {
			return fmt.Errorf("%s: key is required", where)
		}
		if _, dup := seen[f.Key]; dup {
			return fmt.Errorf("%s: duplicate key", where)
		}
		seen[f.Key] = struct{}{}

		if !slices.Contains(knownTypes, f.Type) {
			return fmt.Errorf("%s: unknown type %q", where, f.Type)
		}
		if (f.Type == TypeSelect || f.Type == TypeMultiSelect) && len(f.Options) == 0 {
			return fmt.Errorf("%s: type %q requires options", where, f.Type)
		}
		if slices.Contains(nestingTypes, f.Type) {
			if len(f.Fields) == 0 {
				return fmt.Errorf("%s: type %q requires nested fields", where, f.Type)
			}
			if err := f.Fields.Validate(); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		} else if len(f.Fields) > 0 {
			return fmt.Errorf("%s: type %q cannot have nested fields", where, f.Type)
		}
	}
	return nil
}

// Field returns the field with the given key, or nil.
func (s Schema) Field(key string) *Field {
	if i := slices.IndexFunc(s, func(f Field) bool { return f.Key == key }); i >= 0 {
		return &s[i]
	}
	return nil
}

// Defaults returns the default value of every field that declares one, applied
// recursively to groups. Repeat fields default to an empty list.
func (s Schema) Defaults() map[string]any {
	out := make(map[string]any)
	for _, f := range s {
		switch f.Type {
		case TypeGroup:
			if nested := f.Fields.Defaults(); len(nested) > 0 {
				out[f.Key] = nested
			}
		case TypeRepeat:
			out[f.Key] = []any{}
		default:
			if f.Default != nil {
				out[f.Key] = f.Default
			}
		}
	}
	return out
}
