package index_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/reader"
)

func newProject(t *testing.T) (root string, types *content.Registry) {
	t.Helper()
	return t.TempDir(), content.DefaultRegistry()
}

func post(id, title, slug string, extra ...string) string {
	var b strings.Builder
	b.WriteString("---\nid: " + id + "\ntitle: " + title + "\nslug: " + slug + "\nstatus: published\n")
	for _, line := range extra {
		b.WriteString(line + "\n")
	}
	b.WriteString("---\n\nbody of " + title + "\n")
	return b.String()
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func openIndex(t *testing.T, root string, types *content.Registry) *index.Index {
	t.Helper()
	ix, err := index.Open(root, types)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { ix.Close() })
	return ix
}

const (
	idA = "01J8KQ2P3R4S5T6V7W8X9YZAB1"
	idB = "01J8KQ2P3R4S5T6V7W8X9YZAB2"
	idC = "01J8KQ2P3R4S5T6V7W8X9YZAB3"
)

func TestReconcileIndexesContent(t *testing.T) {
	root, types := newProject(t)
	write(t, root, "content/posts/a/index.md", post(idA, "Alpha", "alpha", "tags: [Go, CMS]"))
	write(t, root, "content/posts/b/index.md", post(idB, "Beta", "beta", "tags: [Go]"))

	ix := openIndex(t, root, types)
	stats, err := ix.Reconcile(t.Context())
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if stats.Scanned != 2 || stats.Indexed != 2 {
		t.Errorf("stats = %+v, want 2 scanned and 2 indexed", stats)
	}

	r := reader.New(ix.DB())
	item, err := r.Get(t.Context(), idA)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.Title != "Alpha" {
		t.Errorf("Title = %q", item.Title)
	}
	if !slices.Equal(item.Terms("tags"), []string{"Go", "CMS"}) {
		t.Errorf("tag terms = %v, want the front matter order preserved", item.Terms("tags"))
	}
}

// The index is a cache: deleting it and rebuilding must reproduce it exactly.
// If this ever fails, something non-derivable has leaked into the database.
func TestRebuildProducesIdenticalRows(t *testing.T) {
	root, types := newProject(t)
	for i, id := range []string{idA, idB, idC} {
		write(t, root, "content/posts/p"+string(rune('a'+i))+"/index.md",
			post(id, "Post", "post-"+string(rune('a'+i)), "tags: [Go]"))
	}
	write(t, root, "content/pages/about.md", post("01J8KQ2P3R4S5T6V7W8X9YZAB4", "About", "about"))

	ix := openIndex(t, root, types)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := dumpRows(t, ix)

	if _, err := ix.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	second := dumpRows(t, ix)

	if first != second {
		t.Errorf("rebuild produced different rows\n--- incremental ---\n%s\n--- rebuilt ---\n%s", first, second)
	}
}

func TestReconcileSkipsUnchangedFiles(t *testing.T) {
	root, types := newProject(t)
	write(t, root, "content/posts/a/index.md", post(idA, "Alpha", "alpha"))

	// Age the file before indexing: the racy timestamp rule re-reads anything
	// modified within a second of the last index write, so a freshly written
	// file is always re-read no matter what stat says.
	old := time.Now().Add(-10 * time.Second)
	p := filepath.Join(root, "content", "posts", "a", "index.md")
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}

	ix := openIndex(t, root, types)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}

	stats, err := ix.Reconcile(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Skipped != 1 || stats.Indexed != 0 {
		t.Errorf("stats = %+v, want the unchanged file skipped", stats)
	}
}

func TestRacyTimestampForcesReread(t *testing.T) {
	root, types := newProject(t)
	p := "content/posts/a/index.md"
	write(t, root, p, post(idA, "Alpha", "alpha"))

	ix := openIndex(t, root, types)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Rewrite the body while keeping size and mtime identical: exactly the
	// case a stat comparison alone cannot detect.
	full := filepath.Join(root, filepath.FromSlash(p))
	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(original), "body of Alpha", "body of ALPHA", 1)
	if len(modified) != len(original) {
		t.Fatalf("test setup changed the file size: %d vs %d", len(modified), len(original))
	}
	if err := os.WriteFile(full, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(full, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	stats, err := ix.Reconcile(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Indexed != 1 {
		t.Errorf("stats = %+v, want the racy file re-read despite matching stat data", stats)
	}
}

func TestReconcileRemovesDeletedFiles(t *testing.T) {
	root, types := newProject(t)
	write(t, root, "content/posts/a/index.md", post(idA, "Alpha", "alpha"))
	write(t, root, "content/posts/b/index.md", post(idB, "Beta", "beta"))

	ix := openIndex(t, root, types)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "content", "posts", "b")); err != nil {
		t.Fatal(err)
	}

	stats, err := ix.Reconcile(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Removed != 1 {
		t.Errorf("stats = %+v, want 1 removed", stats)
	}
	if n, _ := ix.Count(t.Context()); n != 1 {
		t.Errorf("count = %d, want 1", n)
	}
}

func TestFileWithoutIDIsReportedNotIndexed(t *testing.T) {
	root, types := newProject(t)
	write(t, root, "content/posts/a/index.md", "---\ntitle: No ID\n---\nbody\n")

	ix := openIndex(t, root, types)
	stats, err := ix.Reconcile(t.Context())
	if err == nil {
		t.Fatal("expected an error naming the file without an id")
	}
	if !strings.Contains(err.Error(), "fix-ids") {
		t.Errorf("error should point at the remedy: %v", err)
	}
	if stats.Indexed != 0 {
		t.Errorf("a file without an id must not be indexed: %+v", stats)
	}
}

func TestSchemaVersionMismatchDiscardsDatabase(t *testing.T) {
	root, types := newProject(t)
	write(t, root, "content/posts/a/index.md", post(idA, "Alpha", "alpha"))

	ix := openIndex(t, root, types)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := ix.Close(); err != nil {
		t.Fatal(err)
	}

	// Simulate a database written by an older build.
	reopened, err := index.Open(root, types)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.DB().Exec(`UPDATE meta SET value = '999' WHERE key = 'schema_version'`); err != nil {
		t.Fatal(err)
	}
	reopened.Close()

	fresh, err := index.Open(root, types)
	if err != nil {
		t.Fatalf("reopening a stale database should rebuild, not fail: %v", err)
	}
	defer fresh.Close()

	if n, _ := fresh.Count(t.Context()); n != 0 {
		t.Errorf("stale database was not discarded, count = %d", n)
	}
	if _, err := fresh.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if n, _ := fresh.Count(t.Context()); n != 1 {
		t.Errorf("count after reindex = %d, want 1", n)
	}
}

// dumpRows renders the derived tables in a stable order for comparison.
func dumpRows(t *testing.T, ix *index.Index) string {
	t.Helper()
	var b strings.Builder
	for _, q := range []string{
		`SELECT id, kind, slug, title, status, locator, path, revision FROM contents ORDER BY id`,
		`SELECT content_id, taxonomy, term, position FROM terms ORDER BY content_id, taxonomy, position`,
		`SELECT path, size, content_sha256 FROM files ORDER BY path`,
	} {
		rows, err := ix.DB().QueryContext(context.Background(), q)
		if err != nil {
			t.Fatal(err)
		}
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			for i, v := range vals {
				b.WriteString(cols[i])
				b.WriteByte('=')
				b.WriteString(strings.TrimSpace(sprint(v)))
				b.WriteByte(' ')
			}
			b.WriteByte('\n')
		}
		rows.Close()
		b.WriteString("--\n")
	}
	return b.String()
}

func sprint(v any) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}
