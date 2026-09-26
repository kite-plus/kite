// Package plugin loads the plugins a site runs.
//
// A plugin is a directory under plugins/ with a plugin.yaml in it. It can add
// code and files to the pages it covers, which it declares and Kite injects,
// and it can bring a WebAssembly module whose functions run as build hooks.
// Either way it only ever acts through the hook bus, like Kite's own sitemap
// and feed do.
package plugin

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/pack"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/schema"
)

const (
	// Dir holds the installed plugins, relative to the project root.
	Dir = "plugins"
	// ManifestName is the file that makes a directory a plugin.
	ManifestName = "plugin.yaml"
	// APIVersion is the plugin contract this build implements.
	APIVersion = "kite/plugin/v1"
	// AssetsDir holds files a plugin publishes with the site, at
	// plugins/<id>/ in it.
	AssetsDir = "assets"
	// WasmName is the module whose exports run as build hooks.
	WasmName = "plugin.wasm"
)

// The hooks a module can export, named as Extism calls them.
const (
	HookTransformMarkdown = "transform_markdown"
	HookTransformHTML     = "transform_html"
	HookBuildComplete     = "build_complete"
)

var hookNames = []string{HookTransformMarkdown, HookTransformHTML, HookBuildComplete}

// Where injected code goes.
const (
	AtHead = "head"
	AtBody = "body"
)

// PageKinds are the kinds of page an injection can be limited to.
var PageKinds = []string{"home", "single", "list", "taxonomy", "term", "notFound"}

// Author says who made a plugin.
type Author struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// Injection is code a plugin adds to pages.
type Injection struct {
	// At is where it goes: head, before </head>, or body, before </body>.
	At string `yaml:"at"`

	// Pages limits it to kinds of page, such as single; empty is every page.
	Pages []string `yaml:"pages,omitempty"`

	// Kinds limits it to the pages of these content kinds, such as post.
	Kinds []string `yaml:"kinds,omitempty"`

	// When limits it to settings holding these values, as {provider: giscus}.
	// A list stands for any of its values.
	When map[string]any `yaml:"when,omitempty"`

	// HTML is an html/template, given .Settings, .Site and .Page, and asset,
	// which gives the address of one of the plugin's files.
	HTML string `yaml:"html"`
}

// Manifest is a plugin.yaml.
type Manifest struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	APIVersion string `yaml:"apiVersion"`
	Requires   string `yaml:"requires,omitempty"`

	Description string `yaml:"description,omitempty"`
	Author      Author `yaml:"author,omitempty"`
	License     string `yaml:"license,omitempty"`
	Homepage    string `yaml:"homepage,omitempty"`

	Inject []Injection `yaml:"inject,omitempty"`

	// Hooks lists the functions plugin.wasm exports to run during a build.
	Hooks []string `yaml:"hooks,omitempty"`

	Settings schema.Schema `yaml:"settings,omitempty"`
}

// Validate checks a manifest on its own, before anything it names is read.
func (m *Manifest) Validate() error {
	switch {
	case m.ID == "":
		return fmt.Errorf("plugin: id is required in %s", ManifestName)
	case !config.ValidPluginID(m.ID):
		return fmt.Errorf("plugin: id %q is not lowercase words joined by -", m.ID)
	case m.Name == "":
		return fmt.Errorf("plugin %s: name is required", m.ID)
	case m.Version == "":
		return fmt.Errorf("plugin %s: version is required", m.ID)
	case m.APIVersion == "":
		return fmt.Errorf("plugin %s: apiVersion is required", m.ID)
	case m.APIVersion != APIVersion:
		// A plugin written for another contract would half work, which is
		// worse than not loading it.
		return fmt.Errorf("plugin %s declares apiVersion %q, this build implements %q", m.ID, m.APIVersion, APIVersion)
	case len(m.Inject) == 0 && len(m.Hooks) == 0:
		return fmt.Errorf("plugin %s neither injects anything nor declares a hook", m.ID)
	}
	if err := m.Settings.Validate(); err != nil {
		return fmt.Errorf("plugin %s: %w", m.ID, err)
	}
	for i, in := range m.Inject {
		where := fmt.Sprintf("plugin %s: inject %d", m.ID, i+1)
		switch {
		case in.At != AtHead && in.At != AtBody:
			return fmt.Errorf("%s: at is %q, want head or body", where, in.At)
		case strings.TrimSpace(in.HTML) == "":
			return fmt.Errorf("%s: html is empty", where)
		}
		for _, kind := range in.Pages {
			if !slices.Contains(PageKinds, kind) {
				return fmt.Errorf("%s: %q is not a kind of page (want %s)", where, kind, strings.Join(PageKinds, ", "))
			}
		}
		for key := range in.When {
			if m.Settings.Field(key) == nil {
				return fmt.Errorf("%s: when names %q, which is not a setting", where, key)
			}
		}
	}
	seen := make(map[string]bool, len(m.Hooks))
	for _, name := range m.Hooks {
		switch {
		case !slices.Contains(hookNames, name):
			return fmt.Errorf("plugin %s: %q is not a hook (want %s)", m.ID, name, strings.Join(hookNames, ", "))
		case seen[name]:
			return fmt.Errorf("plugin %s: hook %q is declared twice", m.ID, name)
		}
		seen[name] = true
	}
	return nil
}

// Plugin is a loaded plugin.
type Plugin struct {
	Manifest Manifest

	// Root is the plugin's directory.
	Root fs.FS
	// Assets holds the files it publishes, nil when it has none.
	Assets fs.FS
	// Wasm is its module, nil when it declares no hooks.
	Wasm []byte

	// Packs holds its language packs, whose words sit under "plugin".
	Packs pack.Packs

	injections []injection
}

// ReadManifest parses a plugin's manifest without checking it, for saying
// what a plugin is even when it cannot be used.
func ReadManifest(fsys fs.FS) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, ManifestName)
	if err != nil {
		return nil, fmt.Errorf("plugin: read %s: %w", ManifestName, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("plugin: parse %s: %w", ManifestName, err)
	}
	return &m, nil
}

// Load reads the plugin in fsys, which is expected to be the one called id:
// a plugin is found by the name of its directory, so a manifest naming
// another would be installed under one name and configured under the other.
func Load(fsys fs.FS, id string) (*Plugin, error) {
	manifest, err := ReadManifest(fsys)
	if err != nil {
		return nil, err
	}
	m := *manifest
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if m.ID != id {
		return nil, fmt.Errorf("plugin %s: its directory is called %s; the two have to match", m.ID, id)
	}
	if err := theme.Requires("plugin", m.ID, m.Requires, buildinfo.Version); err != nil {
		return nil, err
	}

	p := &Plugin{Manifest: m, Root: fsys}
	if p.Packs, err = pack.Load(fsys, "plugin "+m.ID); err != nil {
		return nil, err
	}
	if info, err := fs.Stat(fsys, AssetsDir); err == nil && info.IsDir() {
		if p.Assets, err = fs.Sub(fsys, AssetsDir); err != nil {
			return nil, err
		}
	}
	if len(m.Hooks) > 0 {
		p.Wasm, err = fs.ReadFile(fsys, WasmName)
		if err != nil {
			return nil, fmt.Errorf("plugin %s declares hooks but has no %s: %w", m.ID, WasmName, err)
		}
	}
	if p.injections, err = compile(m); err != nil {
		return nil, err
	}
	return p, nil
}

// Installed lists the plugins in a project by id, whether or not they load,
// in the order of their names.
func Installed(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, Dir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		info, err := os.Stat(filepath.Join(root, Dir, e.Name()))
		if err != nil || !info.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ids = append(ids, e.Name())
	}
	return ids, nil
}

// Open loads the plugin installed in a project as id.
func Open(root, id string) (*Plugin, error) {
	if !config.ValidPluginID(id) {
		return nil, fmt.Errorf("plugin: %q is not a plugin id", id)
	}
	dir := filepath.Join(root, Dir, id)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("plugin %s is not installed in %s", id, Dir)
	}
	return Load(os.DirFS(dir), id)
}

// Settings resolves what a site stored for the plugin against its schema:
// every field present, with its default where nothing was stored.
func (p *Plugin) Settings(stored map[string]any) map[string]any {
	return p.Manifest.Settings.Resolve(stored)
}

// Localized returns the manifest in a language, from the plugin's pack for
// it. A string the pack does not have stays as the manifest wrote it.
func (p *Plugin) Localized(lang string) Manifest {
	m := p.Manifest
	words := p.Packs.Pick(lang)
	if len(words) == 0 {
		return m
	}
	say := pack.Say(words)
	m.Name = say("plugin.title", m.Name)
	m.Description = say("plugin.description", m.Description)
	m.Settings = pack.Schema(m.Settings, "plugin.settings.", say)
	return m
}
