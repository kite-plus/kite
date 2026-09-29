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
