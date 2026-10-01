package file

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/lock"
)

func TestAnInstallFromAnIndexIsRecordedBesideIt(t *testing.T) {
	root, _, w := newTestProject(t)
	apply := func(op content.Op) content.Result {
		t.Helper()
		res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{op}})
		if err != nil {
			t.Fatalf("Apply %s: %v", op.Describe(), err)
		}
		return res
	}
	read := func() *lock.File {
		t.Helper()
		f, err := lock.Read(root)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	theme := map[string][]byte{"theme.yaml": []byte("name: paper\n"), "layouts/index.html": []byte("<p>hi</p>\n")}
	origin := &content.Origin{
		Version:  "1.0.0",
		Source:   "https://example.com/index.json",
		Resolved: "https://example.com/paper-1.0.0.zip",
		Checksum: "sha256:" + strings.Repeat("a", 64),
	}

	res := apply(content.PutTheme{Name: "paper", Files: theme, Origin: origin})
	if !slices.Contains(res.Written, lock.Name) {
		t.Errorf("Written = %v, want %s among it", res.Written, lock.Name)
	}
	e, ok := read().Get("theme", "paper")
	if !ok || e.Version != "1.0.0" || e.Source != origin.Source || e.Resolved != origin.Resolved ||
		e.Checksum != origin.Checksum || e.Tree != lock.Tree(theme) || e.Granted != nil {
		t.Fatalf("recorded %+v", e)
	}
	if tree, err := lock.TreeOf(filepath.Join(root, "themes", "paper")); err != nil || tree != e.Tree {
		t.Errorf("the files written are not the files recorded: %s, %v", tree, err)
	}

	grant := &content.Grant{Inject: 1, Loads: []string{"cdn.example.com"}}
	apply(content.PutPlugin{ID: "hello", Files: map[string][]byte{"plugin.yaml": []byte("id: hello\n")},
		Origin: &content.Origin{Version: "0.1.0", Source: origin.Source, Granted: grant}})
	if e, _ := read().Get("plugin", "hello"); e.Granted == nil || e.Granted.Inject != 1 ||
		!slices.Equal(e.Granted.Loads, grant.Loads) || e.Granted.Hooks == nil {
		t.Errorf("the plugin's grant was recorded as %+v", e.Granted)
	}

	// An archive installed over it came from nowhere the lock can name.
	res = apply(content.PutTheme{Name: "paper", Files: theme, Replace: true})
	if _, ok := read().Get("theme", "paper"); ok || !slices.Contains(res.Written, lock.Name) {
		t.Errorf("the theme's origin outlived an install from an archive: %v", res.Written)
	}
	apply(content.DeleteTheme{Name: "paper"})

	// The last record taken away takes the file with it.
	res = apply(content.DeletePlugin{ID: "hello"})
	if _, err := os.Stat(filepath.Join(root, lock.Name)); !errors.Is(err, os.ErrNotExist) || !slices.Contains(res.Removed, lock.Name) {
		t.Errorf("the empty lock was kept: %v, removed %v", err, res.Removed)
	}
	res = apply(content.PutTheme{Name: "paper", Files: theme})
	if _, err := os.Stat(filepath.Join(root, lock.Name)); !errors.Is(err, os.ErrNotExist) || slices.Contains(res.Written, lock.Name) {
		t.Error("an install from an archive wrote a lock")
	}
}

func TestABrokenLockStopsAnInstallRatherThanBeingOverwritten(t *testing.T) {
	root, _, w := newTestProject(t)
	writeFile(t, root, lock.Name, "themes: [\n")
	_, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{content.PutTheme{
		Name: "paper", Files: map[string][]byte{"theme.yaml": []byte("name: paper\n")},
	}}})
	if err == nil || !strings.Contains(err.Error(), lock.Name) || !strings.Contains(err.Error(), "delete it") {
		t.Errorf("err = %v", err)
	}
	if got := readFile(t, root, lock.Name); got != "themes: [\n" {
		t.Errorf("the lock was overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "themes", "paper")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the theme was written before the lock was found broken: %v", err)
	}
}

func TestAPinnedReleaseKeepsTheLockWithoutPackages(t *testing.T) {
	root, _, w := newTestProject(t)
	apply := func(op content.Op) content.Result {
		t.Helper()
		res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{op}})
		if err != nil {
			t.Fatalf("Apply %s: %v", op.Describe(), err)
		}
		return res
	}
	res := apply(content.PinKite{Version: "0.1.7", Checksums: "sha256:" + strings.Repeat("b", 64)})
	f, err := lock.Read(root)
	if err != nil || f.Kite == nil || f.Kite.Version != "0.1.7" || !slices.Contains(res.Written, lock.Name) {
		t.Fatalf("pinned %+v, %v, written %v", f, err, res.Written)
	}

	apply(content.PutPlugin{ID: "hello", Files: map[string][]byte{"plugin.yaml": []byte("id: hello\n")},
		Origin: &content.Origin{Version: "0.1.0", Source: "https://example.com/index.json"}})
	apply(content.DeletePlugin{ID: "hello"})
	if f, err := lock.Read(root); err != nil || f.Kite == nil || f.Kite.Version != "0.1.7" || len(f.Plugins) != 0 {
		t.Errorf("after the last package went: %+v, %v", f, err)
	}
	if _, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{content.PinKite{}}}); !errors.Is(err, content.ErrInvalid) {
		t.Errorf("pinning no version: %v", err)
	}
}
