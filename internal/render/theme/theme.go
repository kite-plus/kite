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

	t := &Theme{Manifest: m, Layouts: layouts}
	if sub, err := fs.Sub(fsys, "assets"); err == nil {
		t.Assets = sub
	}
	if sub, err := fs.Sub(fsys, "static"); err == nil {
		t.Static = sub
	}
	return t, nil
}
