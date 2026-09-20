package reader_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/reader"
)

// fixture builds a project with n posts, newest first by published date.
func fixture(t *testing.T, n int, decorate func(i int) string) *reader.Reader {
	t.Helper()
	root := t.TempDir()
	for i := range n {
		id := fmt.Sprintf("01J8KQ2P3R4S5T6V7W8X9YZ%03d", i)
		body := fmt.Sprintf("---\nid: %s\ntitle: Post %02d\nslug: post-%02d\nstatus: published\npublished_at: 2026-01-%02dT00:00:00Z\n%s---\n\nbody %02d\n",
			id, i, i, i+1, decorate(i), i)
		p := filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i), "index.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ix, err := index.Open(root, content.DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	return reader.New(ix.DB())
}

func noExtra(int) string { return "" }

func titles(page content.Page[content.Summary]) []string {
	out := make([]string, 0, len(page.Items))
	for _, s := range page.Items {
		out = append(out, s.Title)
	}
	return out
}

func TestQueryDefaultOrderIsNewestFirst(t *testing.T) {
	r := fixture(t, 5, noExtra)

	page, err := r.Query(t.Context(), content.Query{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []string{"Post 04", "Post 03", "Post 02", "Post 01", "Post 00"}
	if !slices.Equal(titles(page), want) {
		t.Errorf("order = %v, want %v", titles(page), want)
	}
	if page.HasMore {
		t.Error("HasMore should be false when everything fits on one page")
	}
}

func TestCursorPaginationCoversEveryItemExactlyOnce(t *testing.T) {
	const total = 25
	r := fixture(t, total, noExtra)

	var seen []string
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > total {
			t.Fatal("pagination did not terminate")
		}
		page, err := r.Query(t.Context(), content.Query{Limit: 4, Cursor: cursor})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		seen = append(seen, titles(page)...)
		if !page.HasMore {
			break
		}
		cursor = page.NextCursor
	}

	if len(seen) != total {
		t.Fatalf("saw %d items, want %d", len(seen), total)
	}
	unique := slices.Clone(seen)
	slices.Sort(unique)
	unique = slices.Compact(unique)
	if len(unique) != total {
		t.Errorf("pagination repeated items: %d unique of %d", len(unique), total)
	}
}

// The reason for cursors rather than offsets: a set that changes between page
// requests must not cause items to repeat or vanish.
func TestCursorIsStableWhenItemsAreInsertedMidPagination(t *testing.T) {
	const total = 12
	root := t.TempDir()
	writePost := func(i int) {
		id := fmt.Sprintf("01J8KQ2P3R4S5T6V7W8X9YZ%03d", i)
		body := fmt.Sprintf("---\nid: %s\ntitle: Post %02d\nslug: post-%02d\nstatus: published\npublished_at: 2026-01-%02dT00:00:00Z\n---\n\nbody\n",
			id, i, i, i+1)
		p := filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i), "index.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i := range total {
		writePost(i)
	}

	ix, err := index.Open(root, content.DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	r := reader.New(ix.DB())

	first, err := r.Query(t.Context(), content.Query{Limit: 4})
	if err != nil {
		t.Fatal(err)
	}

	// A new item lands at the top of the ordering between the two requests.
	// With offsets this would push one item from page one onto page two and it
	// would be returned twice.
	writePost(total)
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}

	second, err := r.Query(t.Context(), content.Query{Limit: 4, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}

	for _, title := range titles(second) {
		if slices.Contains(titles(first), title) {
			t.Errorf("%q appeared on both pages after an insert", title)
		}
	}
	want := []string{"Post 07", "Post 06", "Post 05", "Post 04"}
	if !slices.Equal(titles(second), want) {
		t.Errorf("second page = %v, want %v", titles(second), want)
	}
}

func TestQueryFiltersByTerm(t *testing.T) {
	r := fixture(t, 6, func(i int) string {
		if i%2 == 0 {
			return "tag: [Go, Even]\n"
		}
		return "tag: [Go, Odd]\n"
	})

	page, err := r.Query(t.Context(), content.Query{
		TermsAny: map[string][]string{"tag": {"Even"}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(page.Items) != 3 {
		t.Errorf("got %d items, want 3: %v", len(page.Items), titles(page))
	}

	both, err := r.Query(t.Context(), content.Query{
		TermsAll: map[string][]string{"tag": {"Go", "Odd"}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(both.Items) != 3 {
		t.Errorf("TermsAll got %d items, want 3: %v", len(both.Items), titles(both))
	}
}

func TestQueryFiltersByStatusAndText(t *testing.T) {
	r := fixture(t, 4, func(i int) string {
		if i == 0 {
			return "status: draft\n"
		}
		return ""
	})

	drafts, err := r.Query(t.Context(), content.Query{Statuses: []content.Status{content.StatusDraft}})
	if err != nil {
		t.Fatal(err)
	}
	// The fixture writes status twice for i == 0; the later key wins, which is
	// what a reader should observe.
	if len(drafts.Items) > 1 {
		t.Errorf("draft filter returned %d items: %v", len(drafts.Items), titles(drafts))
	}

	hit, err := r.Query(t.Context(), content.Query{Text: "Post 02"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hit.Items) != 1 || hit.Items[0].Title != "Post 02" {
		t.Errorf("text search returned %v", titles(hit))
	}
}

func TestCountTerms(t *testing.T) {
	r := fixture(t, 6, func(i int) string {
		if i%3 == 0 {
			return "tag: [Go]\n"
		}
		return "tag: [Go, Extra]\n"
	})

	counts, err := r.CountTerms(t.Context(), "tag", content.Query{})
	if err != nil {
		t.Fatalf("CountTerms: %v", err)
	}
	got := map[string]int{}
	for _, c := range counts {
		got[c.Term] = c.Count
	}
	if got["Go"] != 6 || got["Extra"] != 4 {
		t.Errorf("counts = %v, want Go=6 Extra=4", got)
	}
	// Most used first keeps tag clouds stable.
	if counts[0].Term != "Go" {
		t.Errorf("counts should be ordered by usage, got %v", counts)
	}
}

func TestGetBySlug(t *testing.T) {
	r := fixture(t, 3, noExtra)
	item, err := r.GetBySlug(t.Context(), "post", "post-01")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if item.Title != "Post 01" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.Body.Raw == "" {
		t.Error("Get should return the body; only summaries omit it")
	}

	if _, err := r.GetBySlug(t.Context(), "post", "nope"); err == nil {
		t.Error("expected ErrNotFound for a missing slug")
	}
}

func TestUnknownSortFieldIsRejected(t *testing.T) {
	r := fixture(t, 1, noExtra)
	_, err := r.Query(t.Context(), content.Query{Sort: []content.SortKey{{Field: "nonsense"}}})
	if err == nil {
		t.Fatal("expected an error for an unknown sort field")
	}
	if !errors.Is(err, content.ErrUnsupportedQuery) {
		t.Errorf("error should be ErrUnsupportedQuery, got %v", err)
	}
}

func TestLimitIsClamped(t *testing.T) {
	r := fixture(t, 3, noExtra)
	page, err := r.Query(t.Context(), content.Query{Limit: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 {
		t.Errorf("got %d items", len(page.Items))
	}
}

func TestSummariesCarryTerms(t *testing.T) {
	r := fixture(t, 2, func(int) string { return "tag: [Go, CMS]\ncategory: [Tech]\n" })
	page, err := r.Query(t.Context(), content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if got := page.Items[0].Taxonomies["tag"]; !slices.Equal(got, []string{"Go", "CMS"}) {
		t.Errorf("tag = %v", got)
	}
	if got := page.Items[0].Taxonomies["category"]; !slices.Equal(got, []string{"Tech"}) {
		t.Errorf("category = %v", got)
	}
}
