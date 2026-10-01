package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/lock"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/site"
)

// ClientFor is the client of the index a site installs from: index, as
// apps.index or KITE_APPS_URL names it, or else Kite's own, signed with key,
// as apps.key or KITE_APPS_KEY names it, or else Kite's own; a mirror of
// Kite's index needs no key of its own. What it fetches is kept in the
// site's .kite/cache/apps, or for no site, root "", in the user's cache.
func ClientFor(root, index, key string) (*Client, error) {
	urls := Indexes
	switch {
	case index != "":
		urls = []string{index}
	case os.Getenv("KITE_APPS_URL") != "":
		urls = []string{os.Getenv("KITE_APPS_URL")}
	}
	if key == "" {
		key = os.Getenv("KITE_APPS_KEY")
	}
	if key == "" {
		key = Key
	}
	pub, err := ParseKey(key)
	if err != nil {
		return nil, fmt.Errorf("the key of the index: %w", err)
	}
	cache := ""
	if root != "" {
		cache = filepath.Join(root, ".kite", "cache", "apps")
	} else if dir, err := os.UserCacheDir(); err == nil {
		cache = filepath.Join(dir, "kite", "apps")
	}
	return NewClient(urls, cache, pub), nil
}

// Dir is where a theme or a plugin, kind, is installed, relative to the
// project root.
func Dir(kind, name string) string {
	if kind == "plugin" {
		return filepath.Join(plugin.Dir, name)
	}
	return filepath.Join(site.ThemesDir, name)
}

// CheckTheme checks a theme's files the way a site checks a theme it loads,
// and that it can be installed under the name it gives itself.
func CheckTheme(files map[string][]byte) (*theme.Theme, error) {
	added, err := theme.Load(archive.FS(files))
	if err != nil {
		return nil, err
	}
	if !added.Manifest.SupportsStatic() {
		return nil, errors.New("the theme says it cannot be built into a static site")
	}
	name := added.Manifest.Name
	switch {
	case name == site.BuiltinTheme:
		return nil, fmt.Errorf("the name %s belongs to the theme built into Kite; the theme needs a name of its own", name)
	case !content.ValidThemeName(name):
		return nil, fmt.Errorf("the theme is named %q, and a name has to be usable as a directory: letters, digits, dots, - and _", name)
	}
	return added, nil
}

// CheckPlugin checks a plugin's files the way a site checks a plugin it
// loads.
func CheckPlugin(files map[string][]byte) (*plugin.Plugin, error) {
	manifest, err := plugin.ReadManifest(archive.FS(files))
	if err != nil {
		return nil, err
	}
	return plugin.Load(archive.FS(files), manifest.ID)
}

// GrantOf is what a plugin does to a site, which its owner is told before it
// runs.
func GrantOf(pl *plugin.Plugin) content.Grant {
	hooks := slices.Clone(pl.Manifest.Hooks)
	if hooks == nil {
		hooks = []string{}
	}
	loads := pl.Hosts()
	if loads == nil {
		loads = []string{}
	}
	return content.Grant{Inject: len(pl.Manifest.Inject), Loads: loads, Hooks: hooks}
}

// Exceeds lists what a plugin's grant asks for beyond what was granted: each
// new site and hook, and more code injected.
func Exceeds(asked, granted content.Grant) []string {
	return lock.Grant{Inject: asked.Inject, Loads: asked.Loads, Hooks: asked.Hooks}.Exceeds(
		lock.Grant{Inject: granted.Inject, Loads: granted.Loads, Hooks: granted.Hooks})
}

// Package is a release fetched from the index and checked the way an
// install checks what it installs, with the origin kite.lock records.
type Package struct {
	App     *App
	Release *Release
	Files   map[string][]byte
	// Theme or Plugin is the package as a site loads it.
	Theme  *theme.Theme
	Plugin *plugin.Plugin
	Origin *content.Origin
}

// Op is the change that installs the package, over an installed one of its
// name when replace.
func (p *Package) Op(replace bool) content.Op {
	if p.Plugin != nil {
		return content.PutPlugin{ID: p.App.ID, Files: p.Files, Replace: replace, Origin: p.Origin}
	}
	return content.PutTheme{Name: p.App.ID, Files: p.Files, Replace: replace, Origin: p.Origin}
}

// Fetch downloads a release and checks that it is the package the index
// says it is.
func (c *Client) Fetch(ctx context.Context, app *App, rel *Release) (*Package, error) {
	data, err := c.Archive(ctx, rel)
	if err != nil {
		return nil, err
	}
	kind, manifest, maxSize, maxFiles := "theme", theme.ManifestName, int64(theme.MaxSize), theme.MaxFiles
	if app.Kind == "plugin" {
		kind, manifest, maxSize, maxFiles = "plugin", plugin.ManifestName, plugin.MaxSize, plugin.MaxFiles
	}
	files, problem := archive.Unpack(data, kind, manifest, maxSize, maxFiles)
	if problem != "" {
		return nil, fmt.Errorf("%s %s %s: %s", app.Kind, app.ID, rel.Version, problem)
	}
	p := &Package{App: app, Release: rel, Files: files, Origin: &content.Origin{
		Version:  rel.Version,
		Source:   c.Source(),
		Resolved: rel.Archive.URLs[0],
		Checksum: lock.Checksum(rel.Archive.SHA256),
	}}
	var name, version string
	if app.Kind == "plugin" {
		if p.Plugin, err = CheckPlugin(files); err != nil {
			return nil, err
		}
		name, version = p.Plugin.Manifest.ID, p.Plugin.Manifest.Version
		g := GrantOf(p.Plugin)
		p.Origin.Granted = &g
	} else {
		if p.Theme, err = CheckTheme(files); err != nil {
			return nil, err
		}
		name, version = p.Theme.Manifest.Name, p.Theme.Manifest.Version
	}
	if name != app.ID || version != rel.Version {
		return nil, fmt.Errorf("the archive of %s %s %s holds %s %s instead", app.Kind, app.ID, rel.Version, name, version)
	}
	return p, nil
}

// Installed is a theme or a plugin a site has in themes/ or plugins/.
type Installed struct {
	Kind, Name, Version, Homepage string
	// Grant is what an installed plugin does to the site.
	Grant *content.Grant
}

// Tracked is an installed package whose updates come from the index:
// kite.lock records it as installed from there, or it was installed some
// other way and its homepage is the repository the index names.
type Tracked struct {
	Installed
	App *App
	// Entry is its record in kite.lock, nil for one installed some other
	// way.
	Entry *lock.Entry
}

// Track finds, among a site's packages, those the index at source is where
// their updates come from. A package of a listed name that came from
// another index, or is another project, is not one of them.
func Track(pkgs []Installed, ix *Index, lf *lock.File, source string) []Tracked {
	var out []Tracked
	for _, pkg := range pkgs {
		app := ix.Find(pkg.Kind, pkg.Name)
		if app == nil {
			continue
		}
		t := Tracked{Installed: pkg, App: app}
		if e, ok := lf.Get(pkg.Kind, pkg.Name); ok {
			if e.Source != source {
				continue
			}
			t.Entry = &e
		} else if !app.ProjectOf(pkg.Homepage) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Status is how an installed package stands against the index.
type Status struct {
	// Latest is a newer version that works with this Kite.
	Latest string
	// Yanked says the installed version is not to be installed any more.
	Yanked bool
	// Delisted says why the package is no longer listed.
	Delisted string
	// Problem says why no version works with this Kite.
	Problem string
}

// Status compares the installed version with the index, for Kite version
// kite.
func (t Tracked) Status(kite string) Status {
	st := Status{Delisted: t.App.Delisted}
	if rel := t.App.Release(t.Version); rel != nil {
		st.Yanked = rel.Yanked
	}
	if st.Delisted != "" {
		return st
	}
	rel, err := t.App.Pick("", kite)
	switch {
	case err != nil:
		st.Problem = err.Error()
	case theme.CompareVersions(rel.Version, t.Version) > 0:
		st.Latest = rel.Version
	}
	return st
}

// Changed says how an installed package's files, whose digest is tree, are
// not what was installed, or "" when they are: for one installed from the
// index, the files kite.lock recorded; for one installed some other way,
// the same version's archive in the index. A package with nothing to
// compare with counts as changed, since nothing says it is not.
func (c *Client) Changed(ctx context.Context, t Tracked, tree string) (string, error) {
	if t.Entry != nil {
		if tree == t.Entry.Tree {
			return "", nil
		}
		return "its files have changed since " + t.Version + " was installed", nil
	}
	rel := t.App.Release(t.Version)
	if rel == nil {
		return "it was installed from an archive of " + t.Version + ", which the index does not list", nil
	}
	same, err := c.Fetch(ctx, t.App, rel)
	if err != nil {
		return "", err
	}
	if lock.Tree(same.Files) == tree {
		return "", nil
	}
	return "its files differ from " + t.Version + " in the index", nil
}

// Granted is what an installed plugin was agreed to do: its record in
// kite.lock, or what it does now when it has none.
func (t Tracked) Granted() content.Grant {
	if t.Entry != nil && t.Entry.Granted != nil {
		g := t.Entry.Granted
		return content.Grant{Inject: g.Inject, Loads: g.Loads, Hooks: g.Hooks}
	}
	if t.Grant != nil {
		return *t.Grant
	}
	return content.Grant{}
}
