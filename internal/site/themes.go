package site

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/themes"
)

// ThemesDir holds the themes a project has installed, one directory each.
const ThemesDir = "themes"

// BuiltinTheme is the name that chooses the theme built into Kite.
const BuiltinTheme = "default"

var themeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidThemeName reports whether a name can stand for a theme: one directory
// under themes/, never a path that leads anywhere else.
func ValidThemeName(name string) bool { return themeName.MatchString(name) }

// Installed is a theme a project could use.
type Installed struct {
	// Name is what theme.name in kite.yaml says to choose it: a directory
	// under themes/, or BuiltinTheme for the theme built into Kite.
	Name    string
	Builtin bool

	// Theme is the loaded theme, nil when it cannot be used. Problem then
	// says why, and Manifest holds what could be read of it, if anything.
	Theme    *theme.Theme
	Manifest *theme.Manifest
	Problem  string
}

// Themes lists the themes a project can choose from: the one built into
// Kite, then each directory of themes/ in name order.
//
// A theme that cannot be used is listed too, with the reason, since a theme
// that has quietly vanished from a list is harder to fix than one that says
// what is wrong with it.
func Themes(root string) []Installed {
	out := []Installed{inspect(BuiltinTheme, true, themes.Default())}

	entries, err := os.ReadDir(filepath.Join(root, ThemesDir))
	if err != nil {
		return out
	}
	for _, entry := range entries {
		name := entry.Name()
		dir := filepath.Join(root, ThemesDir, name)
		// A symbolic link to a theme kept elsewhere is followed.
		if info, err := os.Stat(dir); err != nil || !info.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		one := inspect(name, false, os.DirFS(dir))
		switch {
		case name == BuiltinTheme:
			one.Theme = nil
			one.Problem = "the name default always chooses the theme built into Kite, so this directory is never used; rename it to choose it"
		case !ValidThemeName(name):
			one.Theme = nil
			one.Problem = "a theme's directory name may hold only letters, digits, dots, - and _"
		}
		out = append(out, one)
	}
	return out
}

func inspect(name string, builtin bool, fsys fs.FS) Installed {
	one := Installed{Name: name, Builtin: builtin}
	one.Manifest, _ = theme.ReadManifest(fsys)
	th, err := theme.Load(fsys)
	switch {
	case err != nil:
		one.Problem = err.Error()
	case !th.Manifest.SupportsStatic():
		one.Problem = "the theme says it cannot be built into a static site"
	default:
		one.Theme = th
		one.Manifest = &th.Manifest
	}
	return one
}

// LoadTheme loads the theme a name chooses, refusing one that cannot be used
// for the reason Themes would give.
func LoadTheme(root, name string) (*theme.Theme, error) {
	fsys, origin, err := themeFS(root, name)
	if err != nil {
		return nil, err
	}
	th, err := theme.Load(fsys)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", origin, err)
	}
	if !th.Manifest.SupportsStatic() {
		return nil, fmt.Errorf("%s: the theme says it cannot be built into a static site", origin)
	}
	return th, nil
}

// Variant is the site as it would be with another theme or other settings,
// for trying them before they are saved.
type Variant struct {
	// Theme names the theme as theme.name would; empty keeps the site's own.
	Theme string
	// Settings stands in for the stored theme settings when it is not nil.
	Settings map[string]any
	// BaseURL stands in for the site's address, so that every link a page
	// makes leads to the variant rather than to the site.
	BaseURL string
}

// With returns the site as a variant has it. It shares this site's project
// and index, and nothing is written.
func (s *Site) With(v Variant) (*Site, error) {
	cfg := *s.Config
	if v.Theme != "" {
		cfg.Theme.Name = v.Theme
	}
	if v.Settings != nil {
		cfg.Theme.Settings = v.Settings
	}
	if v.BaseURL != "" {
		cfg.Site.BaseURL = strings.TrimRight(v.BaseURL, "/")
	}
	out, err := assemble(s.Project, &cfg, s.Index)
	if err != nil {
		return nil, err
	}
	out.Problems = s.Problems
	return out, nil
}
