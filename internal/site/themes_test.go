package site_test

import (
	"context"
	"fmt"
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

// A theme says its words in the site's language, and a site says any of them
// its own way with a pack of its own, as i18n/zh-CN.yaml.
func TestASitesPackSaysTheThemesWordsItsOwnWay(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml": "site:\n  title: T\n  baseURL: https://example.com\n  language: zh-CN\n",
		"content/posts/one/index.md": "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ001\ntitle: One\nstatus: published\n" +
			"published_at: 2026-01-02T00:00:00Z\n---\n\nHi.\n",
		"i18n/zh-CN.yaml": "recent: 近作\n",
	})
	s, err := site.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	out := filepath.Join(t.TempDir(), "public")
	if _, _, err := s.Build(context.Background(), site.BuildOptions{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	home, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{">近作</h1>", ">归档</a>", "1 月 2 日"} {
		if !strings.Contains(string(home), want) {
			t.Errorf("the home page does not say %q", want)
		}
	}
}

// A theme says how its listings page, and a site's build.pagination stands in
// for it kind by kind; the rest page by build.pageSize.
func TestAListingPagesAsTheThemeAndThenTheSiteSay(t *testing.T) {
	root := t.TempDir()
	post := func(n int) string {
		return fmt.Sprintf("---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ00%d\ntitle: Post %d\nstatus: published\n"+
			"published_at: 2026-01-0%dT00:00:00Z\ntags: [Go]\n---\n\nHi.\n", n, n, n)
	}
	write(t, root, map[string]string{
		"kite.yaml": "site:\n  title: T\n  baseURL: https://example.com\ntheme:\n  name: paper\n" +
			"build:\n  pageSize: 1\n  pagination: {list: 2}\n",
		"content/posts/one/index.md":         post(1),
		"content/posts/two/index.md":         post(2),
		"content/posts/three/index.md":       post(3),
		"themes/paper/theme.yaml":            paper + "pagination: {home: 0, list: 0}\n",
		"themes/paper/layouts/single.html":   "{{ .Page.Title }}",
		"themes/paper/layouts/list.html":     "{{ len .Pages }}/{{ .Paginator.TotalPages }}",
		"themes/paper/layouts/home.html":     "{{ len .Pages }}/{{ .Paginator.TotalPages }}",
		"themes/paper/layouts/term.html":     "{{ len .Pages }}/{{ .Paginator.TotalPages }}",
		"themes/paper/layouts/taxonomy.html": "{{ len .Terms }}",
	})
	s, err := site.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	out := filepath.Join(t.TempDir(), "public")
	_, files, err := s.Build(context.Background(), site.BuildOptions{OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]string{
		"index.html":                "3/1", // the theme's
		"posts/index.html":          "2/2", // the site's, over the theme's
		"tags/go/page/3/index.html": "1/3", // build.pageSize
	} {
		if data, err := os.ReadFile(filepath.Join(out, file)); err != nil || string(data) != want {
			t.Errorf("%s = %q, %v; want %q (files %v)", file, data, err, want, files)
		}
	}

	write(t, root, map[string]string{"themes/paper/theme.yaml": paper + "pagination: {archive: 0}\n"})
	if _, err := site.LoadTheme(root, "paper"); err == nil || !strings.Contains(err.Error(), "pagination") {
		t.Errorf("a theme paging a kind of listing that does not exist loaded: %v", err)
	}
}

// A site declares a kind of its own in kite.yaml: its items have addresses,
// a listing in the order the kind is read in, neighbors in that order, the
// templates a site gives the kind, a taxonomy of their own, and a place in
// the feed when the kind asks for one.
func TestASiteDeclaresAKindOfItsOwn(t *testing.T) {
	root := t.TempDir()
	project := func(slug string, weight int) string {
		return fmt.Sprintf("---\nid: 01J8KQ2P3R4S5T6V7W8X9YZP%02d\ntitle: %s\nstatus: published\n"+
			"published_at: 2026-01-%02dT00:00:00Z\nweight: %d\nstack: [Go]\nrepo: https://example.com/%s\n---\n\n%s.\n",
			weight, slug, 10+weight, weight, slug, slug)
	}
	write(t, root, map[string]string{
		"kite.yaml": `site: {title: T, baseURL: https://example.com}
content:
  types:
    - kind: project
      label: Project
      dir: projects
      order: weight
      feed: true
      taxonomies: [stack]
      fields:
        - {key: repo, type: url, label: Repository}
`,
		"content/projects/kite/index.md":    project("kite", 1),
		"content/projects/vane/index.md":    project("vane", 3),
		"content/projects/almanac/index.md": project("almanac", 2),
		"layouts/project/single.html": `{{ define "main" }}<a class="repo" href="{{ .Page.Params.repo }}">{{ .Page.Title }}</a>` +
			`{{ with .Page.Prev }}prev:{{ .Title }}{{ end }} {{ with .Page.Next }}next:{{ .Title }}{{ end }}{{ end }}`,
		"layouts/project/list.html": `{{ define "main" }}{{ range .Pages }}{{ .Title }};{{ end }}{{ end }}`,
	})
	s, err := site.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if got := s.Project.Types.Get("project"); got == nil || got.Route != "/projects/:slug" {
		t.Fatalf("the declared kind is %+v", got)
	}
	out := filepath.Join(t.TempDir(), "public")
	if _, _, err := s.Build(context.Background(), site.BuildOptions{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	read := func(file string) string {
		data, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for file, want := range map[string]string{
		"projects/index.html":         "kite;almanac;vane;",
		"projects/almanac/index.html": `<a class="repo" href="https://example.com/almanac">almanac</a>prev:kite next:vane`,
		"stack/go/index.html":         "almanac",
		"rss.xml":                     "<title>vane</title>",
	} {
		if got := read(file); !strings.Contains(got, want) {
			t.Errorf("%s does not hold %q", file, want)
		}
	}
}
