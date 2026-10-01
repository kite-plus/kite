package lock

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestALockKeepsWhatItDoesNotKnow(t *testing.T) {
	f, err := Parse([]byte("lockfileVersion: 1\nkite:\n  version: 0.2.0\nthemes:\n  paper:\n    version: 1.0.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	f.Set("plugin", "search", Entry{Version: "0.1.0", Source: "https://example.com/index.json", Granted: &Grant{Inject: 2}})
	if !f.Delete("theme", "paper") || f.Delete("theme", "paper") {
		t.Error("deleting a recorded theme, then again")
	}
	data, err := f.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "# Written by Kite") || !strings.Contains(text, "kite:\n  version: 0.2.0\n") ||
		strings.Contains(text, "paper") || !strings.Contains(text, "loads: []") {
		t.Errorf("written as\n%s", text)
	}
	again, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := again.Get("plugin", "search")
	if !ok || e.Version != "0.1.0" || e.Granted == nil || e.Granted.Inject != 2 || again.Empty() {
		t.Errorf("read back %+v", again)
	}
	if b, _ := again.Bytes(); string(b) != text {
		t.Errorf("the same records wrote other bytes:\n%s", b)
	}
}

func TestALockOfALaterFormatIsRefused(t *testing.T) {
	if _, err := Parse([]byte("lockfileVersion: 2\n")); err == nil || !strings.Contains(err.Error(), "upgrade Kite") {
		t.Errorf("Parse = %v", err)
	}
	f, err := Read(t.TempDir())
	if err != nil || !f.Empty() || f.LockfileVersion != Version {
		t.Errorf("a project with no lock: %+v, %v", f, err)
	}
}

func TestTheTreeIsTheFilesAsInstalled(t *testing.T) {
	files := map[string][]byte{"theme.yaml": []byte("name: paper\n"), "layouts/index.html": []byte("<p>\r\nhi\r\n</p>\n")}
	tree := Tree(files)
	if !strings.HasPrefix(tree, "sha256:") || len(tree) != len("sha256:")+64 {
		t.Fatalf("tree %q", tree)
	}
	same := map[string][]byte{
		"theme.yaml":         []byte("name: paper\n"),
		"layouts/index.html": []byte("<p>\nhi\n</p>\n"),
		".DS_Store":          []byte("finder"),
		"layouts/Thumbs.db":  []byte("explorer"),
	}
	if Tree(same) != tree {
		t.Error("line endings or what a computer adds changed the tree")
	}
	for name, change := range map[string]map[string][]byte{
		"an edit":   {"theme.yaml": []byte("name: paper\nversion: 2\n"), "layouts/index.html": files["layouts/index.html"]},
		"a rename":  {"theme.yaml": files["theme.yaml"], "layouts/home.html": files["layouts/index.html"]},
		"a removal": {"theme.yaml": files["theme.yaml"]},
	} {
		if Tree(change) == tree {
			t.Errorf("%s left the tree as it was", name)
		}
	}

	dir := t.TempDir()
	for name, data := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := TreeOf(dir); err != nil || got != tree {
		t.Errorf("TreeOf = %q, %v; want %q", got, err, tree)
	}
}

func TestAGrantKnowsWhatGoesBeyondIt(t *testing.T) {
	granted := Grant{Inject: 1, Loads: []string{"cdn.example.com"}, Hooks: []string{"build_complete"}}
	if more := (Grant{Inject: 1, Loads: []string{"cdn.example.com"}}).Exceeds(granted); len(more) != 0 {
		t.Errorf("less than was granted: %v", more)
	}
	more := Grant{Inject: 3, Loads: []string{"cdn.example.com", "evil.example.net"}, Hooks: []string{"build_complete", "transform_markdown"}}.Exceeds(granted)
	want := []string{"loads from evil.example.net", "runs transform_markdown", "injects 3 pieces of code, not 1"}
	if !slices.Equal(more, want) {
		t.Errorf("more = %q, want %q", more, want)
	}
}
