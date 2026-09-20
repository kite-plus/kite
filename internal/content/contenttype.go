package content

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/kite-plus/kite/internal/schema"
)

// Layout says how an item of a given kind is stored on disk.
type Layout string

const (
	// LayoutBundle stores an item as a directory holding index.md plus its
	// media. This is the default: it keeps an item self contained, so copying
	// or deleting the directory moves or removes the whole item.
	LayoutBundle Layout = "bundle"

	// LayoutSingleFile stores an item as one markdown file.
	LayoutSingleFile Layout = "single"
)

// TemplateHints names the templates a kind prefers. The theme engine uses them
// as the "type" dimension of its lookup chain.
type TemplateHints struct {
	Single string
	List   string
}

// Type describes one kind of content. Routing, template lookup and admin form
// generation all read their rules from here rather than switching on Kind, so
// that opening custom types to plugins later is additive.
type Type struct {
	Kind       Kind
	Label      string
	Dir        string // relative to the content root
	Route      string // e.g. "/posts/:slug"
	Layout     Layout
	Fields     schema.Schema
	Taxonomies []string
	Templates  TemplateHints
	Sortable   []string
}

// Validate checks that the type description is usable.
func (t *Type) Validate() error {
	if t.Kind == "" {
		return fmt.Errorf("content type: kind is required")
	}
	if t.Dir == "" {
		return fmt.Errorf("content type %s: dir is required", t.Kind)
	}
	if t.Route == "" {
		return fmt.Errorf("content type %s: route is required", t.Kind)
	}
	if t.Layout != LayoutBundle && t.Layout != LayoutSingleFile {
		return fmt.Errorf("content type %s: unknown layout %q", t.Kind, t.Layout)
	}
	if err := t.Fields.Validate(); err != nil {
		return fmt.Errorf("content type %s: %w", t.Kind, err)
	}
	return nil
}

// Registry holds the known content types.
//
// Registration is internal for now: v1 ships only the built-in post and page
// types. The registry exists from the start so that routing, template lookup
// and form generation never hard-code a kind, which is what makes opening it
// to configuration and plugins later an additive change.
type Registry struct {
	mu    sync.RWMutex
	types map[Kind]*Type
	order []Kind
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{types: make(map[Kind]*Type)}
}

func (r *Registry) register(t *Type) error {
	if err := t.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.types[t.Kind]; exists {
		return fmt.Errorf("content type %s: already registered", t.Kind)
	}
	r.types[t.Kind] = t
	r.order = append(r.order, t.Kind)
	return nil
}

// Get returns the type for a kind, or nil.
func (r *Registry) Get(k Kind) *Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[k]
}

// Kinds returns the registered kinds in registration order.
func (r *Registry) Kinds() []Kind {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.order)
}

// Types returns every registered type in registration order.
func (r *Registry) Types() []*Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Type, 0, len(r.order))
	for _, k := range r.order {
		out = append(out, r.types[k])
	}
	return out
}

// ByDir returns the type stored in the given content subdirectory, or nil.
func (r *Registry) ByDir(dir string) *Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.types {
		if t.Dir == dir {
			return t
		}
	}
	return nil
}

// TaxonomyNames returns every taxonomy referenced by any registered type,
// sorted so that build output stays deterministic.
func (r *Registry) TaxonomyNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := make(map[string]struct{})
	for _, t := range r.types {
		for _, name := range t.Taxonomies {
			set[name] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(set))
}

// DefaultRegistry returns the built-in types: post and page.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, t := range builtinTypes() {
		if err := r.register(t); err != nil {
			panic("content: built-in type registration failed: " + err.Error())
		}
	}
	return r
}

func builtinTypes() []*Type {
	return []*Type{
		{
			Kind:       "post",
			Label:      "Post",
			Dir:        "posts",
			Route:      "/posts/:slug",
			Layout:     LayoutBundle,
			Taxonomies: []string{"tags", "categories"},
			Templates:  TemplateHints{Single: "single", List: "list"},
			Sortable:   []string{"published_at", "updated_at", "title"},
			Fields: schema.Schema{
				{Key: "description", Type: schema.TypeText, Label: "Description"},
				{Key: "cover", Type: schema.TypeImage, Label: "Cover image"},
				{Key: "pinned", Type: schema.TypeBoolean, Label: "Pinned", Default: false},
			},
		},
		{
			Kind:      "page",
			Label:     "Page",
			Dir:       "pages",
			Route:     "/:slug",
			Layout:    LayoutSingleFile,
			Templates: TemplateHints{Single: "single", List: "list"},
			Sortable:  []string{"updated_at", "title"},
			Fields: schema.Schema{
				{Key: "description", Type: schema.TypeText, Label: "Description"},
				{Key: "menu_weight", Type: schema.TypeNumber, Label: "Menu weight"},
			},
		},
	}
}
