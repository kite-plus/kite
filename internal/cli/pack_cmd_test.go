package cli

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// A theme packs as a release carries it: what the theme is made of, under a
// folder named after it, and nothing that lives beside it in its repository.
// The same files pack to the same bytes, and the archive installs.
func TestAThemePacksAsAReleaseCarriesIt(t *testing.T) {
	work := t.TempDir()
	runKite(t, work, "theme", "new", "paper")
	dir := filepath.Join(work, "paper")
	writeFiles(t, dir, map[string]string{
		"LICENSE":                  "MIT\n",
		"README.md":                "# Paper\n",
		"notes.txt":                "not part of the theme\n",
		".DS_Store":                "x",
		"node_modules/tool/a.js":   "x",
		"layouts/.draft.html":      "x",
		"example/kite.yaml":        "site: {title: Example}\n",
		"static/.cache/stale.css":  "x",
		"i18n/zh-CN.yaml":          "posts: 文章\n",
		"layouts/_partials/a.html": "<p>a</p>\n",
	})

	runKite(t, dir, "theme", "pack")
	first := filepath.Join(dir, "dist", "paper-0.1.0.zip")
	want := []string{
		"paper/LICENSE", "paper/README.md", "paper/i18n/en.yaml", "paper/i18n/zh-CN.yaml",
		"paper/layouts/404.html", "paper/layouts/_partials/a.html", "paper/layouts/baseof.html",
		"paper/layouts/list.html", "paper/layouts/single.html", "paper/layouts/taxonomy.html",
		"paper/static/style.css", "paper/theme.yaml",
	}
	if got := zipNames(t, first); !slices.Equal(got, want) {
		t.Errorf("archive holds\n  %q\nwant\n  %q", got, want)
	}

	second := filepath.Join(t.TempDir(), "again.zip")
	out := runKite(t, dir, "theme", "pack", "--output", second, "--json")
	var report packed
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("pack --json said %q: %v", out, err)
	}
	if report.Name != "paper" || report.Version != "0.1.0" || report.Files != len(want) || len(report.SHA256) != 64 {
		t.Errorf("report = %+v", report)
	}
	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Error("the same theme packed twice made different bytes")
	}

	site := newSite(t)
	if out := runKite(t, site, "theme", "add", first); !strings.Contains(out, "installed paper 0.1.0") {
		t.Errorf("add said:\n%s", out)
	}
}

// A plugin packs the same way, its module and assets with it.
func TestAPluginPacksAsAReleaseCarriesIt(t *testing.T) {
	work := t.TempDir()
	runKite(t, work, "plugin", "new", "greet")
	dir := filepath.Join(work, "greet")
	writeFiles(t, dir, map[string]string{"LICENSE": "MIT\n", "main.go": "package main\n"})

	out := runKite(t, dir, "plugin", "pack")
	if !strings.Contains(out, "packed greet 0.1.0") {
		t.Errorf("pack said:\n%s", out)
	}
	want := []string{"greet/LICENSE", "greet/assets/greet.css", "greet/plugin.yaml"}
	if got := zipNames(t, filepath.Join(dir, "dist", "greet-0.1.0.zip")); !slices.Equal(got, want) {
		t.Errorf("archive holds %q, want %q", got, want)
	}
}

// A theme with a file that is not a plain one would be refused by every
// install, so it is not packed.
func TestALinkInAThemeIsNotPacked(t *testing.T) {
	work := t.TempDir()
	runKite(t, work, "theme", "new", "paper")
	dir := filepath.Join(work, "paper")
	if err := os.Symlink(filepath.Join(dir, "layouts", "list.html"), filepath.Join(dir, "layouts", "home.html")); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	if err := runKiteErr(t, dir, "theme", "pack"); !strings.Contains(err.Error(), "not a plain file") {
		t.Errorf("pack = %v, want a refusal of the link", err)
	}
}
