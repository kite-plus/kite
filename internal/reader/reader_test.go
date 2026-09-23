package reader_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

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
			return "tags: [Go, Even]\n"
		}
		return "tags: [Go, Odd]\n"
	})

	page, err := r.Query(t.Context(), content.Query{
		TermsAny: map[string][]string{"tags": {"Even"}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(page.Items) != 3 {
		t.Errorf("got %d items, want 3: %v", len(page.Items), titles(page))
	}

	both, err := r.Query(t.Context(), content.Query{
		TermsAll: map[string][]string{"tags": {"Go", "Odd"}},
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
			return "tags: [Go]\n"
		}
		return "tags: [Go, Extra]\n"
	})

	counts, err := r.CountTerms(t.Context(), "tags", content.Query{})
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
	r := fixture(t, 2, func(int) string { return "tags: [Go, CMS]\ncategories: [Tech]\n" })
	page, err := r.Query(t.Context(), content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if got := page.Items[0].Taxonomies["tags"]; !slices.Equal(got, []string{"Go", "CMS"}) {
		t.Errorf("tag = %v", got)
	}
	if got := page.Items[0].Taxonomies["categories"]; !slices.Equal(got, []string{"Tech"}) {
		t.Errorf("category = %v", got)
	}
}

// A build decides in SQL what a site publishes, and the domain decides it in
// Go. The day the two disagree, a post is public in one place and not in the
// other.
func TestPublicAtAgreesWithIsPublic(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	items := []struct{ slug, front string }{
		{"published-past", "status: published\npublished_at: 2026-01-01T00:00:00Z\n"},
		{"published-now", "status: published\npublished_at: 2026-06-01T12:00:00Z\n"},
		{"published-next-second", "status: published\npublished_at: 2026-06-01T12:00:01Z\n"},
		{"published-future", "status: published\npublished_at: 2027-01-01T00:00:00Z\n"},
		{"published-undated", "status: published\n"},
		{"hugo-future", "date: 2027-01-01T00:00:00Z\n"},
		{"scheduled-past", "status: scheduled\npublished_at: 2026-05-31T12:00:00Z\n"},
		{"scheduled-now", "status: scheduled\npublished_at: 2026-06-01T12:00:00Z\n"},
		{"scheduled-next-second", "status: scheduled\npublished_at: 2026-06-01T12:00:01Z\n"},
		{"scheduled-undated", "status: scheduled\n"},
		{"draft", "status: draft\npublished_at: 2026-01-01T00:00:00Z\n"},
		{"archived", "status: archived\npublished_at: 2026-01-01T00:00:00Z\n"},
		{"deleted", "status: published\npublished_at: 2026-01-01T00:00:00Z\ndeleted_at: 2026-02-01T00:00:00Z\n"},
	}

	root := t.TempDir()
	for i, it := range items {
		body := fmt.Sprintf("---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ%03d\ntitle: %s\nslug: %s\n%s---\n\nbody\n", i, it.slug, it.slug, it.front)
		p := filepath.Join(root, "content", "posts", it.slug, "index.md")
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
	r := reader.New(ix.DB())

	// Deleted items are asked for too, so that it is the public filter
	// itself that has to leave them out.
	page, err := r.Query(t.Context(), content.Query{PublicAt: &now, IncludeDeleted: true, Limit: content.MaxLimit})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var public []string
	for _, s := range page.Items {
		public = append(public, s.Slug)
	}
	slices.Sort(public)

	for _, it := range items {
		item, err := r.GetBySlug(t.Context(), "post", it.slug)
		if err != nil {
			t.Fatalf("GetBySlug(%s): %v", it.slug, err)
		}
		if want, got := item.IsPublic(now), slices.Contains(public, it.slug); got != want {
			t.Errorf("%s: selected = %v, but IsPublic = %v", it.slug, got, want)
		}
	}

	want := []string{"published-now", "published-past", "published-undated", "scheduled-now", "scheduled-past"}
	if !slices.Equal(public, want) {
		t.Errorf("public = %v, want %v", public, want)
	}

	count, err := r.Count(t.Context(), content.Query{PublicAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	if count != len(want) {
		t.Errorf("Count = %d, want %d", count, len(want))
	}
}

// A build loads every item through GetMany, so what it returns has to be
// what Get would have, item for item.
func TestGetManyReturnsWhatGetDoesInTheOrderAsked(t *testing.T) {
	r := fixture(t, 7, func(i int) string {
		if i%2 == 0 {
			return "tags: [Go, Notes]\n"
		}
		return ""
	})

	ids := []content.ID{
		"01J8KQ2P3R4S5T6V7W8X9YZ005", "01J8KQ2P3R4S5T6V7W8X9YZ000",
		"01J8KQ2P3R4S5T6V7W8X9YZ999", "01J8KQ2P3R4S5T6V7W8X9YZ002",
	}
	items, err := r.GetMany(t.Context(), ids)
	if err != nil {
		t.Fatalf("GetMany: %v", err)
	}
	var got []content.ID
	for _, item := range items {
		got = append(got, item.ID)
	}
	if want := []content.ID{ids[0], ids[1], ids[3]}; !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v: in the order asked, without the one that does not exist", got, want)
	}

	for _, item := range items {
		one, err := r.Get(t.Context(), item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(item, one) {
			t.Errorf("%s differs from Get:\n got %+v\nwant %+v", item.ID, item, one)
		}
	}
}

// A list shows which posts are pinned. Only a real true pins one, and the
// index has to say so exactly when the item itself does.
func TestSummariesSayWhichItemsArePinned(t *testing.T) {
	pins := []string{"pinned: true\n", "pinned: false\n", "pinned: \"yes\"\n", "pinned: 1\n", ""}
	r := fixture(t, len(pins), func(i int) string { return pins[i] })

	page, err := r.Query(t.Context(), content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	var pinned []string
	for _, s := range page.Items {
		item, err := r.Get(t.Context(), s.ID)
		if err != nil {
			t.Fatal(err)
		}
		if s.Pinned != item.Summarize().Pinned {
			t.Errorf("%s: the list says pinned=%v, the item says %v", s.Title, s.Pinned, item.Summarize().Pinned)
		}
		if s.Pinned {
			pinned = append(pinned, s.Title)
		}
	}
	if want := []string{"Post 00"}; !slices.Equal(pinned, want) {
		t.Errorf("pinned = %v, want %v", pinned, want)
	}
}
