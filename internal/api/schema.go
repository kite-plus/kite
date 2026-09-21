package api

import (
	"reflect"
	"slices"
	"strings"
	"time"
)

// jsonSchema is the subset of JSON Schema an OpenAPI 3.1 document needs.
type jsonSchema struct {
	Ref         string                 `json:"$ref,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Description string                 `json:"description,omitempty"`
	Items       *jsonSchema            `json:"items,omitempty"`
	Properties  map[string]*jsonSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`

	// AdditionalProperties carries the value schema of a map.
	AdditionalProperties *jsonSchema `json:"additionalProperties,omitempty"`

	// Nullable is expressed as a type union, which is how 3.1 spells it.
	Types []string `json:"-"`
}

var timeType = reflect.TypeFor[time.Time]()

// schemaFor derives a schema from a Go type, registering every named struct it
// meets in components.
//
// Deriving rather than declaring is the point: the wire types are the
// contract, so a field added to one appears in the document without anybody
// remembering to write it down. A hand-kept document is a document that
// eventually describes something the server no longer does.
func schemaFor(t reflect.Type, components map[string]*jsonSchema) *jsonSchema {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t == timeType {
		return &jsonSchema{Type: "string", Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.Bool:
		return &jsonSchema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &jsonSchema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &jsonSchema{Type: "number"}
	case reflect.String:
		return &jsonSchema{Type: "string"}
	case reflect.Slice, reflect.Array:
		return &jsonSchema{Type: "array", Items: schemaFor(t.Elem(), components)}
	case reflect.Map:
		return &jsonSchema{Type: "object", AdditionalProperties: schemaFor(t.Elem(), components)}
	case reflect.Interface:
		// A meta value is whatever the author put in the front matter.
		return &jsonSchema{}
	case reflect.Struct:
		name := schemaName(t)
		if _, done := components[name]; !done {
			// Reserved before recursing so a self-referential type
			// terminates.
			components[name] = &jsonSchema{}
			*components[name] = *structSchema(t, components)
		}
		return &jsonSchema{Ref: "#/components/schemas/" + name}
	default:
		return &jsonSchema{}
	}
}

func structSchema(t reflect.Type, components map[string]*jsonSchema) *jsonSchema {
	out := &jsonSchema{Type: "object", Properties: map[string]*jsonSchema{}}
	collectFields(t, out, components)
	if len(out.Required) > 1 {
		slices.Sort(out.Required)
	}
	return out
}

// collectFields walks a struct, flattening embedded ones the way encoding/json
// does.
func collectFields(t reflect.Type, out *jsonSchema, components map[string]*jsonSchema) {
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")

		if f.Anonymous && name == "" {
			embedded := f.Type
			for embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct && embedded != timeType {
				collectFields(embedded, out, components)
				continue
			}
		}
		if name == "" {
			name = f.Name
		}

		out.Properties[name] = schemaFor(f.Type, components)
		// A field that is never omitted is always present, which is what
		// "required" means to a client generator.
		if !strings.Contains(opts, "omitempty") && !strings.Contains(opts, "omitzero") &&
			f.Type.Kind() != reflect.Pointer {
			out.Required = append(out.Required, name)
		}
	}
}

// schemaName turns a Go type into a component name, spelling a generic
// instantiation the way a client would want to read it: List[Summary] becomes
// SummaryList.
func schemaName(t reflect.Type) string {
	name := t.Name()
	base, args, ok := strings.Cut(name, "[")
	if !ok {
		return name
	}
	arg := strings.TrimSuffix(args, "]")
	// reflect spells a type argument with its full import path.
	if i := strings.LastIndex(arg, "."); i >= 0 {
		arg = arg[i+1:]
	}
	return arg + base
}
