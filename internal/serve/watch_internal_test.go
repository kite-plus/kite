package serve

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A theme linked into themes/, as one being written in its own repository
// is, is watched where it lives, since a walk does not follow the link.
func TestAThemeLinkedIntoThemesIsWatched(t *testing.T) {
	project, elsewhere := t.TempDir(), t.TempDir()
	layouts := filepath.Join(elsewhere, "layouts")
	for _, dir := range []string{layouts, filepath.Join(project, "themes")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(elsewhere, filepath.Join(project, "themes", "paper")); err != nil {
		t.Skipf("cannot link a theme here: %v", err)
	}
	w, err := newWatcher(project, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })

	if err := os.WriteFile(filepath.Join(layouts, "single.html"), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-w.fsw.Events:
			if filepath.Base(event.Name) == "single.html" && w.interesting(event) {
				return
			}
		case err := <-w.fsw.Errors:
			t.Fatal(err)
		case <-deadline:
			t.Fatal("an edit to a template of a linked theme went unnoticed")
		}
	}
}

// A theme is fingerprinted by what it is assembled from, so a site kept in
// the theme's own repository, writing its index as it runs, does not make
// the theme look edited.
func TestOnlyWhatAThemeIsMadeOfIsFingerprinted(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "themes", "paper")
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(themeDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("theme.yaml", "name: paper\n")
	write("layouts/single.html", "first")
	write("example/.kite/index.db", "first")

	s := &Server{root: root}
	before := s.templatesStamp("paper")

	write("example/.kite/index.db", "written again, and longer")
	if s.templatesStamp("paper") != before {
		t.Error("a file the theme is not made of changed its fingerprint")
	}

	write("layouts/single.html", "edited, and longer")
	if s.templatesStamp("paper") == before {
		t.Error("an edited template left the fingerprint as it was")
	}
}
