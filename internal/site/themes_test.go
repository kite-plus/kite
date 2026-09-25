package site_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/site"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const paper = `name: paper
title: Paper
version: 1.2.0
apiVersion: kite/v1
settings:
  - {key: accent, type: color, default: "#2563eb"}
  - {key: columns, type: number, default: 1}
`

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml": "site:\n  title: T\n  baseURL: https://example.com\n" +
			"theme:\n  name: default\n  settings:\n    accent: \"#a3473b\"\n    nav: \"About | /about/\"\n",
		"content/pages/about.md": "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ001\ntitle: About\nstatus: published\n---\nHi.\n",

		"themes/paper/theme.yaml":            paper,
		"themes/paper/layouts/single.html":   "{{ .Site.ThemeSettings.accent }}",
		"themes/future/theme.yaml":           "name: future\ntitle: Future\nversion: 1.0.0\napiVersion: kite/v9\n",
		"themes/future/layouts/single.html":  "x",
		"themes/notes/README.md":             "not a theme",
		"themes/default/theme.yaml":          paper,
		"themes/default/layouts/single.html": "x",
	})
	return root
}

// Every theme a project has is listed, the ones it cannot use with the
// reason, since a theme that silently went missing from the list would be
// harder to put right than one that says what is wrong with it.
func TestThemesListsWhatAProjectHas(t *testing.T) {
	got := map[string]site.Installed{}
	var order []string
	for _, one := range site.Themes(project(t)) {
		got[one.Name] = one
		if !one.Builtin {
			order = append(order, one.Name)
		}
	}

	if builtin := site.Themes(project(t))[0]; !builtin.Builtin || builtin.Name != site.BuiltinTheme || builtin.Theme == nil {
		t.Errorf("first = %+v, want the built-in theme, usable", builtin)
	}
	if strings.Join(order, " ") != "default future notes paper" {
		t.Errorf("installed = %v, want the directories in name order", order)
	}
	if paper := got["paper"]; paper.Theme == nil || paper.Problem != "" || paper.Manifest.Title != "Paper" {
		t.Errorf("paper = %+v, want it usable", paper)
	}
	if future := got["future"]; future.Theme != nil || !strings.Contains(future.Problem, "kite/v9") ||
		future.Manifest == nil || future.Manifest.Title != "Future" {
		t.Errorf("future = %+v, want it refused for its contract, and still named", future)
	}
	if notes := got["notes"]; notes.Theme != nil || notes.Problem == "" {
		t.Errorf("notes = %+v, want a directory with no theme.yaml said to be unusable", notes)
	}
	if shadowed := got["default"]; shadowed.Builtin || shadowed.Theme != nil || !strings.Contains(shadowed.Problem, "rename") {
		t.Errorf("themes/default = %+v, want it said to be hidden by the built-in theme", shadowed)
	}
}

func TestAThemeNameIsNeverAPath(t *testing.T) {
	root := project(t)
	for _, name := range []string{"../paper", "paper/..", "/etc", ".hidden", "a b"} {
		if _, err := site.LoadTheme(root, name); err == nil {
			t.Errorf("LoadTheme(%q) loaded something", name)
		}
	}
	if th, err := site.LoadTheme(root, "paper"); err != nil || th.Manifest.Name != "paper" {
		t.Errorf("LoadTheme(paper) = %v, %v", th, err)
	}
}

// A variant is how a theme or a setting is tried before it is saved: the
// site it returns draws with them, and the site it came from does not change.
func TestAVariantTriesAThemeWithoutSavingIt(t *testing.T) {
	s, err := site.Open(context.Background(), project(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	tried, err := s.With(site.Variant{
		Theme:    "paper",
		Settings: map[string]any{"columns": "3"},
		BaseURL:  "http://127.0.0.1:1717/preview/abc/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tried.Theme.Manifest.Name != "paper" {
		t.Errorf("theme = %s", tried.Theme.Manifest.Name)
	}
	if settings := tried.ThemeSettings(); settings["columns"] != float64(3) || settings["accent"] != "#2563eb" {
		t.Errorf("settings = %v, want the variant's read as the theme declares them", settings)
	}
	if link := tried.Resolver.ForHome("en"); !strings.HasPrefix(link, "/preview/abc/") {
		t.Errorf("home = %s, want it under the variant's address", link)
	}

	if s.Theme.Manifest.Name != "default" || s.Config.Theme.Name != "default" {
		t.Errorf("the site itself changed to %s", s.Theme.Manifest.Name)
	}
	settings := s.ThemeSettings()
	if settings["accent"] != "#a3473b" {
		t.Errorf("accent = %v", settings["accent"])
	}
}
