package theme

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
)

// Source is one place templates come from. Sources are consulted in order, so
// a site's layouts directory precedes the theme's.
type Source struct {
	Name string
	FS   fs.FS
}

// Engine resolves templates and renders pages.
type Engine struct {
	sources []Source
	formats map[string]Format
	funcs   template.FuncMap

	mu    sync.RWMutex
	cache map[string]*template.Template
}

// Options configures an engine.
type Options struct {
	// Sources are searched in order; put the site's own layouts first.
	Sources []Source

	// Formats are the output formats a target may ask for.
	Formats map[string]Format

	// Funcs are merged over the built-in namespaces, for values that are only
	// known once a site is loaded.
	Funcs template.FuncMap
}

// NewEngine returns an engine.
func NewEngine(opts Options) *Engine {
	formats := opts.Formats
	if formats == nil {
		formats = map[string]Format{FormatHTML.Name: FormatHTML}
	}

	funcs := baseFuncs()
	for name, fn := range opts.Funcs {
		funcs[name] = fn
	}

	return &Engine{
		sources: slices.Clone(opts.Sources),
		formats: formats,
		funcs:   funcs,
		cache:   make(map[string]*template.Template),
	}
}

// Lookup reports which template a target resolves to, and the full chain that
// was searched.
//
// The chain is returned even on success so that `kite doctor` can explain a
// resolution. "Why is my template not being used" is the most common question
// a theme author asks, and answering it with a command is cheaper than
// answering it in documentation.
func (e *Engine) Lookup(t Target) (Candidate, []string, bool) {
	candidates := Candidates(t, e.formats)
	found, ok := resolve(e.sources, candidates)
	return found, candidates, ok
}

// Render executes the template for a target against data.
func (e *Engine) Render(t Target, data any) ([]byte, error) {
	tmpl, err := e.template(t)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("theme: execute %s: %w", t.Kind, err)
	}
	return buf.Bytes(), nil
}

// HasTemplate reports whether a target resolves to anything.
func (e *Engine) HasTemplate(t Target) bool {
	_, _, ok := e.Lookup(t)
	return ok
}

func (e *Engine) template(t Target) (*template.Template, error) {
	key := t.Kind + "|" + t.Type + "|" + t.Layout + "|" + t.Format + "|" + t.Lang

	e.mu.RLock()
	cached, ok := e.cache[key]
	e.mu.RUnlock()
	if ok {
		return cached, nil
	}

	found, chain, ok := e.Lookup(t)
	if !ok {
		return nil, fmt.Errorf("theme: no template for %s (tried %s)", t.Kind, strings.Join(chain, ", "))
	}

	tmpl, err := e.build(t, found)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[key] = tmpl
	e.mu.Unlock()
	return tmpl, nil
}

// build assembles the template set for one page: every partial, the page
// template itself, and the surrounding base template when one exists.
func (e *Engine) build(t Target, page Candidate) (*template.Template, error) {
	root := template.New("kite").Funcs(e.funcs)

	// Partials are bound late so that a partial can call another partial.
	root = root.Funcs(template.FuncMap{
		"partial":       partialFunc(root),
		"partialCached": partialFunc(root),
	})

	if err := e.parsePartials(root); err != nil {
		return nil, err
	}

	pageSrc, err := e.read(page.Path)
	if err != nil {
		return nil, err
	}

	baseTarget := Target{Kind: "baseof", Type: t.Type, Format: t.Format, Lang: t.Lang}
	base, _, hasBase := e.Lookup(baseTarget)
	if !hasBase {
		// No base template: the page template is the whole document.
		if _, err := root.New(page.Path).Parse(string(pageSrc)); err != nil {
			return nil, fmt.Errorf("theme: parse %s: %w", page.Path, err)
		}
		return root.Lookup(page.Path), nil
	}

	baseSrc, err := e.read(base.Path)
	if err != nil {
		return nil, err
	}
	// The base is parsed first so that the page's {{ define "main" }} replaces
	// the base's {{ block "main" }} placeholder.
	if _, err := root.New(base.Path).Parse(string(baseSrc)); err != nil {
		return nil, fmt.Errorf("theme: parse %s: %w", base.Path, err)
	}
	if _, err := root.Lookup(base.Path).Parse(string(pageSrc)); err != nil {
		return nil, fmt.Errorf("theme: parse %s: %w", page.Path, err)
	}
	return root.Lookup(base.Path), nil
}

// parsePartials loads every partial from every source. A partial defined by an
// earlier source wins, which is how a site overrides one piece of a theme
// without copying the rest.
func (e *Engine) parsePartials(root *template.Template) error {
	seen := make(map[string]struct{})
	for _, src := range e.sources {
		err := fs.WalkDir(src.FS, PartialsDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // a source without partials is fine
			}
			if d.IsDir() {
				return nil
			}
			name := strings.TrimPrefix(p, PartialsDir+"/")
			if _, dup := seen[name]; dup {
				return nil
			}
			data, err := fs.ReadFile(src.FS, p)
			if err != nil {
				return err
			}
			if _, err := root.New(name).Parse(string(data)); err != nil {
				return fmt.Errorf("theme: parse partial %s: %w", p, err)
			}
			seen[name] = struct{}{}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) read(name string) ([]byte, error) {
	for _, src := range e.sources {
		data, err := fs.ReadFile(src.FS, name)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("theme: template %s not found", name)
}

// partialFunc renders a named partial into the surrounding document.
func partialFunc(root *template.Template) func(string, any) (template.HTML, error) {
	return func(name string, data any) (template.HTML, error) {
		t := root.Lookup(name)
		if t == nil {
			t = root.Lookup(path.Base(name))
		}
		if t == nil {
			return "", fmt.Errorf("partial %q not found", name)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("partial %q: %w", name, err)
		}
		// The partial was rendered by html/template, so it is already escaped
		// in its own context.
		return template.HTML(buf.String()), nil //nolint:gosec // produced by the template engine
	}
}
