package content

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
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

// Order is how the items of a kind are listed, and read one after another.
type Order string

const (
	// OrderDate lists items newest first, as a blog does, and reads them
	// from the oldest.
	OrderDate Order = "date"

	// OrderWeight lists and reads items by the weight their front matter
	// gives, smallest first, as documentation is read; items with no weight
	// follow, by title.
	OrderWeight Order = "weight"
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

	// Order is how the kind's items are listed and read; empty is OrderDate.
	Order Order

	// Feed says the kind's items are news, which the site's feed carries: a
	// post is, a standalone page such as about is not.
	Feed bool
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
	if t.Order != "" && t.Order != OrderDate && t.Order != OrderWeight {
		return fmt.Errorf("content type %s: unknown order %q (want date or weight)", t.Kind, t.Order)
	}
	if err := t.Fields.Validate(); err != nil {
		return fmt.Errorf("content type %s: %w", t.Kind, err)
	}
	return nil
}

// Registry holds the known content types: the built-in post and page, and
// the kinds a site declares of its own in kite.yaml. Routing, template lookup
// and form generation read them from here and never hard-code a kind.
type Registry struct {
	mu       sync.RWMutex
	types    map[Kind]*Type
	order    []Kind
	declared []Kind
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

var (
	kindName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	dirName  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

// Declare puts the kinds a site declares of its own in the registry, beside
// the built-in ones, in place of those it declared before. Nothing changes
// when one of them cannot be used.
//
// A declared kind is named by lowercase letters, digits, - and _, and keeps
// its items in a folder of its own directly under content/, which is also
// its listing's address: two kinds cannot share one, nor can a kind and a
// taxonomy, whose pages and templates would be at the same place.
func (r *Registry) Declare(declared ...*Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	builtin := make([]*Type, 0, len(r.order))
	for _, k := range r.order {
		if !slices.Contains(r.declared, k) {
			builtin = append(builtin, r.types[k])
		}
	}
	taxonomies := make(map[string]bool)
	for _, t := range slices.Concat(builtin, declared) {
		for _, name := range t.Taxonomies {
			taxonomies[name] = true
		}
	}
	kinds := make(map[Kind]bool)
	dirs := make(map[string]Kind)
	for _, t := range builtin {
		kinds[t.Kind], dirs[t.Dir] = true, t.Kind
	}
	for _, t := range declared {
		if err := t.Validate(); err != nil {
			return err
		}
		switch {
		case !kindName.MatchString(string(t.Kind)):
			return fmt.Errorf("content type %s: a kind is named by lowercase letters, digits, - and _", t.Kind)
		case kinds[t.Kind]:
			return fmt.Errorf("content type %s: the kind is declared twice, or is built in", t.Kind)
		case !dirName.MatchString(t.Dir):
			return fmt.Errorf("content type %s: dir %q is not one folder named by lowercase letters, digits, - and _", t.Kind, t.Dir)
		case dirs[t.Dir] != "":
			return fmt.Errorf("content type %s: content/%s already holds the %s items", t.Kind, t.Dir, dirs[t.Dir])
		case taxonomies[string(t.Kind)] || taxonomies[t.Dir]:
			return fmt.Errorf("content type %s: a taxonomy has the name, and its pages would be at the same address", t.Kind)
		case !slices.Contains(strings.Split(t.Route, "/"), ":slug") || !strings.HasPrefix(t.Route, "/"):
			return fmt.Errorf("content type %s: route %q does not start with / and hold :slug, as /%s/:slug does", t.Kind, t.Route, t.Dir)
		}
		for _, name := range t.Taxonomies {
			if !kindName.MatchString(name) {
				return fmt.Errorf("content type %s: taxonomy %q is not named by lowercase letters, digits, - and _", t.Kind, name)
			}
		}
		kinds[t.Kind], dirs[t.Dir] = true, t.Kind
	}

	r.types = make(map[Kind]*Type, len(builtin)+len(declared))
	r.order, r.declared = nil, nil
	for _, t := range slices.Concat(builtin, declared) {
		r.types[t.Kind] = t
		r.order = append(r.order, t.Kind)
	}
	for _, t := range declared {
		r.declared = append(r.declared, t.Kind)
	}
	return nil
}

// Declared returns the kinds a site declared, in order.
func (r *Registry) Declared() []*Type {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Type, 0, len(r.declared))
	for _, k := range r.declared {
		out = append(out, r.types[k])
	}
	return out
}

// FeedKinds returns the kinds whose items the site's feed carries.
func (r *Registry) FeedKinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for _, k := range r.order {
		if r.types[k].Feed {
			out = append(out, string(k))
		}
	}
	return out
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
			Feed:       true,
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
