package cli

import (
	"archive/zip"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/config"
)

// zipTheme packs a theme directory the way a release does: one folder named
// after the theme.
func zipTheme(t *testing.T, dir, name string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), name+".zip")
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		w, err := zw.Create(name + "/" + filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

// A theme started with kite theme new goes into a site, draws it, and comes
// back out with the commands alone.
func TestAThemeIsAddedUsedAndRemovedFromTheCommandLine(t *testing.T) {
	work := t.TempDir()
	runKite(t, work, "theme", "new", "paper")
	if out := runKite(t, work, "theme", "verify", "paper"); !strings.Contains(out, "identical built and served") {
		t.Errorf("verify said:\n%s", out)
	}

	site := newSite(t)
	writePost(t, site, "01J8KQ2P3R4S5T6V7W8X9YZ000", "hello", "status: published\npublished_at: 2026-01-01T00:00:00Z\n")
	runKite(t, site, "theme", "add", filepath.Join(work, "paper"))
	if err := runKiteErr(t, site, "theme", "add", filepath.Join(work, "paper")); !strings.Contains(err.Error(), "--replace") {
		t.Errorf("adding it twice = %v, want a pointer to --replace", err)
	}
	// A release's archive installs the same theme over the one there.
	runKite(t, site, "theme", "add", "--replace", zipTheme(t, filepath.Join(work, "paper"), "paper"))
	if out := runKite(t, site, "theme", "list"); !strings.Contains(out, "*  default") || !strings.Contains(out, "   paper") {
		t.Errorf("list before switching:\n%s", out)
	}

	runKite(t, site, "theme", "use", "paper")
	cfg, err := config.Load(site)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme.Name != "paper" {
		t.Fatalf("theme.name = %q", cfg.Theme.Name)
	}
	runKite(t, site, "build")
	page, err := os.ReadFile(filepath.Join(site, "public", "posts", "hello", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<link rel="stylesheet" href="/style.css">`) {
		t.Error("the post is not drawn with the new theme")
	}
	if _, err := os.Stat(filepath.Join(site, "public", "style.css")); err != nil {
		t.Errorf("the theme's stylesheet was not published: %v", err)
	}

	if err := runKiteErr(t, site, "theme", "remove", "paper"); !strings.Contains(err.Error(), "in use") {
		t.Errorf("removing the theme in use = %v", err)
	}
	if err := runKiteErr(t, site, "theme", "remove", "default"); !strings.Contains(err.Error(), "built into Kite") {
		t.Errorf("removing the built-in theme = %v", err)
	}
	runKite(t, site, "theme", "use", "default")
	runKite(t, site, "theme", "remove", "paper")
	if _, err := os.Stat(filepath.Join(site, "themes", "paper")); !os.IsNotExist(err) {
		t.Error("the theme's directory is still there")
	}
}

// A theme that cannot be used is listed with the reason, and is not switched
// to.
func TestAThemeThatDoesNotLoadIsNotUsed(t *testing.T) {
	site := newSite(t)
	dir := filepath.Join(site, "themes", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "theme.yaml"), []byte("name: broken\nversion: 0.1.0\napiVersion: kite/v9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runKiteErr(t, site, "theme", "use", "broken"); !strings.Contains(err.Error(), "kite/v9") {
		t.Errorf("use = %v, want the theme's own problem", err)
	}
	if out := runKite(t, site, "theme", "list"); !strings.Contains(out, "kite/v9") {
		t.Errorf("list does not say why it cannot be used:\n%s", out)
	}
	if err := runKiteErr(t, site, "theme", "new", "default"); !strings.Contains(err.Error(), "built into Kite") {
		t.Errorf("new default = %v", err)
	}
}
