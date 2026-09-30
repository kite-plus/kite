package site_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/site"
	"github.com/kite-plus/kite/themes"
)

const post = "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ001\ntitle: Hello\nstatus: published\n---\nHi.\n"

func open(t *testing.T, root string) *site.Site {
	t.Helper()
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// tree lists what a project holds, leaving out the inside of .kite, where
// the index keeps working files of its own.
func tree(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		paths = append(paths, filepath.ToSlash(rel))
		if d.IsDir() && d.Name() == ".kite" {
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// A build swaps its output in whole and deletes what was there, so an output
// that would take the project, or a part of it, along is refused before
// anything is touched.
func TestAnOutputThatWouldTakeTheProjectAlongIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml":                    "site:\n  title: T\n  baseURL: https://example.com\n",
		"content/posts/hello/index.md": post,
		"static/robots.txt":            "User-agent: *\n",
	})
	s := open(t, root)
	before := tree(t, root)

	for _, tc := range []struct{ name, out, says string }{
		{"the project", root, "would replace the project"},
		{"a directory holding it", filepath.Dir(root), "would replace the project"},
		{"its content", filepath.Join(root, "content"), "would replace the project's content"},
		{"the account and index", filepath.Join(root, ".kite"), "would replace the project's .kite"},
		{"the repository", filepath.Join(root, ".git"), "would replace the project's .git"},
		{"a directory the build copies", filepath.Join(root, "static", "site"), "inside the project's static"},
		{"the configuration", filepath.Join(root, "kite.yaml"), "is a file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: tc.out})
			if err == nil || !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("build to %s = %v, want a refusal saying %q", tc.out, err, tc.says)
			}
		})
	}
	if after := tree(t, root); !slices.Equal(after, before) {
		t.Errorf("a refused build changed the project:\n%q\nwas\n%q", after, before)
	}

	if _, _, err := s.Build(t.Context(), site.BuildOptions{}); err != nil {
		t.Fatalf("the default output was refused: %v", err)
	}
}

// The project is the project under any name: through a symlink, or spelled
// in another case on a disk that ignores case.
func TestTheProjectIsKnownUnderAnotherName(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml":                    "site:\n  title: T\n  baseURL: https://example.com\n",
		"content/posts/hello/index.md": post,
	})
	s := open(t, root)

	var names []string
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(filepath.Dir(root), link); err == nil {
		names = append(names, filepath.Join(link, filepath.Base(root)))
	}
	if upper := strings.ToUpper(root); upper != root {
		a, errA := os.Stat(upper)
		b, errB := os.Stat(root)
		if errA == nil && errB == nil && os.SameFile(a, b) {
			names = append(names, upper)
		}
	}
	if len(names) == 0 {
		t.Skip("no other name leads to a directory here")
	}
	for _, out := range names {
		if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: out}); err == nil ||
			!strings.Contains(err.Error(), "would replace the project") {
			t.Errorf("build to %s = %v, want a refusal", out, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "kite.yaml")); err != nil {
		t.Errorf("the project is gone: %v", err)
	}
}

// A theme or a plugin can be linked in from its own repository, outside the
// project. The output may not replace one, but it may sit beside them, or in
// the example site a theme's repository holds.
func TestAnOutputMayNotReplaceALinkedThemeOrPlugin(t *testing.T) {
	dir := t.TempDir()
	themeRepo := filepath.Join(dir, "theme")
	if err := os.CopyFS(themeRepo, themes.Default()); err != nil {
		t.Fatal(err)
	}
	pluginRepo := filepath.Join(dir, "plugin")
	write(t, pluginRepo, map[string]string{
		"plugin.yaml": "id: hello\nname: Hello\nversion: 1.0.0\napiVersion: kite/plugin/v1\n" +
			"inject:\n  - at: head\n    html: <meta name=\"hello\">\n",
	})

	for _, tc := range []struct{ name, root string }{
		{"beside them", filepath.Join(dir, "site")},
		{"in the theme's repository", filepath.Join(themeRepo, "example")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			write(t, tc.root, map[string]string{
				"kite.yaml": "site:\n  title: T\n  baseURL: https://example.com\n" +
					"theme:\n  name: linked\nplugins: [hello]\n",
				"content/posts/hello/index.md": post,
			})
			for link, target := range map[string]string{"themes/linked": themeRepo, "plugins/hello": pluginRepo} {
				at := filepath.Join(tc.root, filepath.FromSlash(link))
				if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, at); err != nil {
					t.Skipf("cannot link a theme here: %v", err)
				}
			}
			s := open(t, tc.root)

			if _, _, err := s.Build(t.Context(), site.BuildOptions{}); err != nil {
				t.Fatalf("the default output was refused: %v", err)
			}
			for _, repo := range []string{themeRepo, pluginRepo} {
				if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: repo}); err == nil ||
					!strings.Contains(err.Error(), "would replace the project") {
					t.Errorf("build to %s = %v, want a refusal", repo, err)
				}
			}
			for _, kept := range []string{filepath.Join(themeRepo, "theme.yaml"), filepath.Join(pluginRepo, "plugin.yaml")} {
				if _, err := os.Stat(kept); err != nil {
					t.Errorf("%s is gone: %v", kept, err)
				}
			}
		})
	}
}

// A build replaces its output whole, so a directory it did not write, such as
// a home folder named by mistake, is refused and keeps every file in it. One
// that holds nothing, or only what a desktop leaves in a folder, is built in.
func TestAnOutputKiteDidNotWriteIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml":                    "site:\n  title: T\n  baseURL: https://example.com\n",
		"content/posts/hello/index.md": post,
	})
	s := open(t, root)

	documents := filepath.Join(t.TempDir(), "Documents")
	write(t, documents, map[string]string{"taxes.pdf": "numbers", "letters/mum.txt": "hi"})
	_, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: documents})
	if err == nil || !strings.Contains(err.Error(), "holds files Kite did not write") {
		t.Fatalf("build over a folder of documents = %v, want a refusal", err)
	}
	if got := tree(t, documents); !slices.Equal(got, []string{".", "letters", "letters/mum.txt", "taxes.pdf"}) {
		t.Errorf("the refused build changed the folder: %q", got)
	}

	shown := filepath.Join(t.TempDir(), "site")
	write(t, shown, map[string]string{".DS_Store": "x", "Thumbs.db": "x"})
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: shown}); err != nil {
		t.Errorf("a folder holding only what a desktop leaves was refused: %v", err)
	}
}

// The output a build wrote is replaced by the next build, and so is one an
// older Kite wrote before builds were recorded, known by its sitemap or its
// feed of this site. Another site's output is not this site's to replace.
func TestAnOutputKiteWroteIsReplaced(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml":                    "site:\n  title: T\n  baseURL: https://example.com/blog/\n",
		"content/posts/hello/index.md": post,
	})
	s := open(t, root)

	out := filepath.Join(t.TempDir(), "public")
	for range 2 {
		if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: out}); err != nil {
			t.Fatalf("building again over the build's own output: %v", err)
		}
	}

	older := filepath.Join(t.TempDir(), "public")
	write(t, older, map[string]string{
		"sitemap.xml": "<?xml version=\"1.0\"?>\n<urlset>\n  <url>\n    <loc>https://example.com/blog/posts/hello/</loc>\n  </url>\n</urlset>\n",
		"old.html":    "gone after the build",
	})
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: older}); err != nil {
		t.Fatalf("an output an older Kite wrote was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(older, "old.html")); !os.IsNotExist(err) {
		t.Errorf("the older output was not replaced: %v", err)
	}

	fed := filepath.Join(t.TempDir(), "public")
	write(t, fed, map[string]string{
		"rss.xml": "<rss version=\"2.0\"><channel><link>https://example.com/blog/</link></channel></rss>",
	})
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: fed}); err != nil {
		t.Errorf("an older output known by its feed was refused: %v", err)
	}

	other := filepath.Join(t.TempDir(), "public")
	write(t, other, map[string]string{
		"sitemap.xml": "<urlset><url><loc>https://another.example/</loc></url></urlset>",
	})
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: other}); err == nil {
		t.Error("another site's output was replaced")
	}
}

// A build stages beside its output under names only Kite gives, so a
// directory beside it of the kind anyone might keep, public.tmp or
// public.prev, is not deleted.
func TestWhatLiesBesideTheOutputIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{
		"kite.yaml":                    "site:\n  title: T\n  baseURL: https://example.com\n",
		"content/posts/hello/index.md": post,
		"public.tmp/notes.txt":         "mine",
		"public.prev/notes.txt":        "mine too",
	})
	s := open(t, root)
	if _, _, err := s.Build(t.Context(), site.BuildOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"public.tmp/notes.txt", "public.prev/notes.txt"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s is gone: %v", name, err)
		}
	}
	for _, name := range []string{"public.kite-stage", "public.kite-previous"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("%s was left behind: %v", name, err)
		}
	}
}
