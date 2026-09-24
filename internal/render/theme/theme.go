// Package theme loads themes and renders pages with them.
//
// The contract this package implements is internal for now: v1 ships one
// built-in theme and the engine is exercised only by Kite itself. It is
// published, versioned and frozen once a second theme has been written against
// it, because a contract whose only user is its own author is a contract with
// undiscovered defects.
package theme

import (
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/schema"
)

// APIVersion is the theme contract version this build implements.
const APIVersion = "kite/v1"

// LayoutsDir is where templates live, in a theme and in a site alike. Sharing
// one name makes the override rule a plain path comparison: the same relative
// path in the site wins over the theme.
const LayoutsDir = "layouts"

// PartialsDir holds non-routable templates.
const PartialsDir = "_partials"

// ManifestName is the theme descriptor.
const ManifestName = "theme.yaml"

// Author describes a theme's author.
type Author struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// Layout is a template a theme offers items to choose by name, as front
// matter writes it: layout: links. It is looked up like any other page
// template, as layouts/<type>/<name>.html and then layouts/<name>.html, and
// is declared so the admin can list it rather than leave it to be guessed.
type Layout struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label,omitempty"`
	// Description says what the layout is for.
	Description string `yaml:"description,omitempty"`
	// Types limits the layout to some content types; left empty it is
	// offered to every type.
	Types []string `yaml:"types,omitempty"`
}

// ForType reports whether the layout is offered to items of a type.
func (l Layout) ForType(kind string) bool {
	return len(l.Types) == 0 || slices.Contains(l.Types, kind)
}

var layoutName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidLayoutName reports whether a name can stand for a layout. It is a
// template's file name, never a path, so front matter cannot point a page at
// a template outside the layouts it is meant to choose from.
func ValidLayoutName(name string) bool { return layoutName.MatchString(name) }

// Manifest is the parsed theme.yaml.
type Manifest struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	APIVersion string `yaml:"apiVersion"`
	Requires   string `yaml:"requires,omitempty"`

	Title       string   `yaml:"title,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Author      Author   `yaml:"author,omitempty"`
	License     string   `yaml:"license,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`

	// Capabilities lists the runtimes the theme supports: static, dynamic.
	Capabilities []string `yaml:"capabilities,omitempty"`

	ContentTypes []string `yaml:"contentTypes,omitempty"`
	Taxonomies   []string `yaml:"taxonomies,omitempty"`

	// Settings drives the theme configuration form in the admin, so that a
	// theme author never writes admin code.
	Settings schema.Schema `yaml:"settings,omitempty"`

	// Layouts are the templates an item may choose besides its type's own.
	Layouts []Layout `yaml:"layouts,omitempty"`
}

// Validate checks a manifest.
//
// An unknown apiVersion is refused rather than rendered on a best-effort
// basis: half-rendering a theme written for a different contract produces
// pages that look subtly wrong, which is worse than a clear failure.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("theme: name is required in %s", ManifestName)
	}
	if m.APIVersion == "" {
		return fmt.Errorf("theme %s: apiVersion is required in %s", m.Name, ManifestName)
	}
	if m.APIVersion != APIVersion {
		return fmt.Errorf("theme %s declares apiVersion %q, this build implements %q",
			m.Name, m.APIVersion, APIVersion)
	}
	if err := m.Settings.Validate(); err != nil {
		return fmt.Errorf("theme %s: %w", m.Name, err)
	}
	seen := make(map[string]bool, len(m.Layouts))
	for _, l := range m.Layouts {
		if !ValidLayoutName(l.Name) {
			return fmt.Errorf("theme %s: layout name %q must be lowercase letters, digits, - or _",
				m.Name, l.Name)
		}
		if seen[l.Name] {
			return fmt.Errorf("theme %s: layout %q is declared twice", m.Name, l.Name)
		}
		seen[l.Name] = true
	}
	return nil
}

// SupportsStatic reports whether the theme can be rendered into a static site.
func (m *Manifest) SupportsStatic() bool { return m.supports("static") }

// SupportsDynamic reports whether the theme can be rendered by a server.
func (m *Manifest) SupportsDynamic() bool { return m.supports("dynamic") }

func (m *Manifest) supports(capability string) bool {
	if len(m.Capabilities) == 0 {
		return true // a theme that says nothing supports both
	}
	for _, c := range m.Capabilities {
		if strings.EqualFold(c, capability) {
			return true
		}
	}
	return false
}

// DefaultSettings returns the theme's declared defaults.
func (m *Manifest) DefaultSettings() map[string]any { return m.Settings.Defaults() }

// Theme is a loaded theme.
type Theme struct {
	Manifest Manifest

	// Layouts is rooted at the theme's layouts directory.
	Layouts fs.FS

	// Assets and Static are optional and may be nil.
	Assets fs.FS
	Static fs.FS
}

// Load reads a theme from a filesystem rooted at the theme directory.
func Load(fsys fs.FS) (*Theme, error) {
	data, err := fs.ReadFile(fsys, ManifestName)
	if err != nil {
		return nil, fmt.Errorf("theme: read %s: %w", ManifestName, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("theme: parse %s: %w", ManifestName, err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := m.Satisfied(buildinfo.Version); err != nil {
		return nil, err
	}

	layouts, err := fs.Sub(fsys, LayoutsDir)
	if err != nil {
		return nil, fmt.Errorf("theme %s: %s directory is missing: %w", m.Name, LayoutsDir, err)
	}
	if err := checkLayouts(&m, layouts); err != nil {
		return nil, err
	}

	t := &Theme{Manifest: m, Layouts: layouts}
	if sub, err := fs.Sub(fsys, "assets"); err == nil {
		t.Assets = sub
	}
	if sub, err := fs.Sub(fsys, "static"); err == nil {
		t.Static = sub
	}
	return t, nil
}

// checkLayouts makes sure every layout a theme offers has a template it can
// be drawn with: in the directory of each type it is offered to, or at the
// top of layouts/ where every type finds it. A layout offered without one
// would quietly fall back to the type's own template, which is a promise the
// admin would be making on the theme's behalf and breaking.
func checkLayouts(m *Manifest, layouts fs.FS) error {
	exists := func(name string) bool {
		_, err := fs.Stat(layouts, name)
		return err == nil
	}
	for _, l := range m.Layouts {
		top := exists(l.Name + ".html")
		if len(l.Types) == 0 {
			if !top {
				return fmt.Errorf("theme %s offers layout %q to every type but has no %s/%s.html",
					m.Name, l.Name, LayoutsDir, l.Name)
			}
			continue
		}
		for _, kind := range l.Types {
			if !top && !exists(kind+"/"+l.Name+".html") {
				return fmt.Errorf("theme %s offers layout %q to %s but has no %s/%s/%s.html or %s/%s.html",
					m.Name, l.Name, kind, LayoutsDir, kind, l.Name, LayoutsDir, l.Name)
			}
		}
	}
	return nil
}
