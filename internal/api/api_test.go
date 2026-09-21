package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/site"
)

const config = `site:
  title: Field Notes
  description: A test project.
  baseURL: https://example.com
  language: en
`

func post(i int, extra string) string {
	status := "published"
	if i%5 == 4 {
		status = "draft"
	}
	return fmt.Sprintf(`---
id: 01J8KQ2P3R4S5T6V7W8X9YZ%03d
title: Post %02d
slug: post-%02d
status: %s
published_at: 2026-01-%02dT00:00:00Z
%s---

Body of post %02d, which mentions marmalade only in this one.
`, i, i, i, status, i+1, extra, i)
}

// newProject writes a project with n posts and one page.
func newProject(t *testing.T, n int) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "kite.yaml"), config)

	for i := range n {
		terms := "tags: [Go]\n"
		if i%2 == 0 {
			terms = "tags: [Go, Notes]\ncategories: [Tech]\n"
		}
		write(t, filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i), "index.md"), post(i, terms))
	}
	write(t, filepath.Join(root, "content", "pages", "about.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ900\ntitle: About\nslug: about\nstatus: published\n---\n\nAbout.\n")
	return root
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newServer opens a project and serves its read model.
func newServer(t *testing.T, root string) (http.Handler, *site.Site) {
	t.Helper()
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatalf("site.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	srv := api.New(api.Options{Site: func() api.View {
		return api.View{
			Reader:   s.Reader,
			Resolver: s.Resolver,
			Types:    s.Project.Types,
			Site:     s.Config.Site,
			Store:    s.Config.Content.Store,
			Runtime:  "test",
			Problems: s.Problems,
		}
	}})

	// Mounted the way a running server mounts it, so the tests exercise the
	// prefix and the routing together.
	mux := http.NewServeMux()
	srv.Mount(mux)
	return mux, s
}

// get issues a request and decodes the body into v, asserting the status.
func get[T any](t *testing.T, h http.Handler, path string, want int) T {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	if rec.Code != want {
		t.Fatalf("GET %s: status = %d, want %d\nbody: %s", path, rec.Code, want, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("GET %s: content type = %q, want JSON", path, ct)
	}

	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("GET %s: decode: %v\nbody: %s", path, err, rec.Body.String())
	}
	return v
}

func TestListIsNewestFirstAndCarriesTheLivePageURL(t *testing.T) {
	h, _ := newServer(t, newProject(t, 5))

	page := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post", http.StatusOK)
	if len(page.Items) != 5 {
		t.Fatalf("got %d posts, want 5", len(page.Items))
	}
	if got, want := page.Items[0].Slug, "post-04"; got != want {
		t.Errorf("first item = %q, want %q (newest first)", got, want)
	}
	if got, want := page.Items[0].URL, "/posts/post-04/"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
	// Where the bytes live is a different question from where the page is.
	if got, want := page.Items[0].Locator, "content/posts/post-04"; got != want {
		t.Errorf("locator = %q, want %q", got, want)
	}
}

// Paging over a set that is being edited is the whole reason there is no
// offset in this API. The guarantee is that a walk sees each item once.
func TestCursorPagingVisitsEveryItemExactlyOnce(t *testing.T) {
	h, _ := newServer(t, newProject(t, 12))

	seen := map[string]int{}
	path := api.Prefix + "/contents?kind=post&limit=5"
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("paging did not terminate")
		}
		page := get[api.List[api.Summary]](t, h, path, http.StatusOK)
		for _, item := range page.Items {
			seen[item.ID]++
		}
		if !page.HasMore {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("has_more is true but no cursor was returned")
		}
		path = api.Prefix + "/contents?kind=post&limit=5&cursor=" + page.NextCursor
	}

	if len(seen) != 12 {
		t.Errorf("saw %d distinct posts, want 12", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("%s appeared %d times, want once", id, n)
		}
	}
}

// A filter that is silently ignored returns a page that looks right and is
// not, which is worse than an error: the client has no way to notice.
func TestMisspelledParameterIsRefused(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	body := get[api.ErrorBody](t, h, api.Prefix+"/contents?stat=draft", http.StatusBadRequest)
	if body.Error.Code != api.CodeInvalidRequest {
		t.Errorf("code = %q, want %q", body.Error.Code, api.CodeInvalidRequest)
	}
	if body.Error.Field != "stat" {
		t.Errorf("field = %q, want %q", body.Error.Field, "stat")
	}
}

func TestBadInputIsFourHundredWithTheFieldNamed(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	for _, tc := range []struct{ query, field string }{
		{"status=nonsense", "status"},
		{"limit=0", "limit"},
		{"limit=abc", "limit"},
		{"cursor=!!!notbase64", "cursor"},
		{"term=missing-colon", "term"},
		{"published_from=yesterday", "published_from"},
		{"include_deleted=perhaps", "include_deleted"},
	} {
		body := get[api.ErrorBody](t, h, api.Prefix+"/contents?"+tc.query, http.StatusBadRequest)
		if body.Error.Field != tc.field {
			t.Errorf("?%s: field = %q, want %q", tc.query, body.Error.Field, tc.field)
		}
	}
}

// The reader refuses a query it cannot answer rather than loading everything
// and filtering in Go. That refusal has to reach the client as a client
// error, not as a server fault.
func TestUnknownSortFieldIsAClientError(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	body := get[api.ErrorBody](t, h, api.Prefix+"/contents?sort=-nonsense", http.StatusBadRequest)
	if body.Error.Code != api.CodeUnsupportedQuery {
		t.Errorf("code = %q, want %q", body.Error.Code, api.CodeUnsupportedQuery)
	}
}

func TestFiltersSelect(t *testing.T) {
	h, _ := newServer(t, newProject(t, 10))

	for _, tc := range []struct {
		name  string
		query string
		want  int
	}{
		{"by kind", "kind=post", 10},
		{"by status", "kind=post&status=draft", 2},
		{"by term", "term=categories:Tech", 5},
		{"by two terms, any", "term=tags:Go&term=tags:Notes", 10},
		{"by two terms, all", "term_all=tags:Go&term_all=tags:Notes", 5},
		{"by text", "q=marmalade", 10},
		{"by text, no match", "q=zzzznope", 0},
		{"by date window", "kind=post&published_from=2026-01-06T00:00:00Z", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?limit=500&"+tc.query, http.StatusOK)
			if len(page.Items) != tc.want {
				t.Errorf("got %d items, want %d", len(page.Items), tc.want)
			}
		})
	}
}

// Counting is a second query over the whole filtered set, so it happens only
// when asked for.
func TestTotalIsOptInAndCountsThePastTheCurrentPage(t *testing.T) {
	h, _ := newServer(t, newProject(t, 10))

	plain := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=3", http.StatusOK)
	if plain.Total != nil {
		t.Errorf("total = %v, want absent unless requested", *plain.Total)
	}

	counted := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=3&count=true", http.StatusOK)
	if counted.Total == nil {
		t.Fatal("total is absent although count=true was asked for")
	}
	if *counted.Total != 10 {
		t.Errorf("total = %d, want 10", *counted.Total)
	}
	if len(counted.Items) != 3 {
		t.Errorf("got %d items, want the requested page of 3", len(counted.Items))
	}

	// The total describes the filtered set, not the whole site.
	drafts := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&status=draft&count=true", http.StatusOK)
	if *drafts.Total != 2 {
		t.Errorf("draft total = %d, want 2", *drafts.Total)
	}
}

func TestItemCarriesBodyAndListingDoesNot(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	page := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post", http.StatusOK)
	id := page.Items[0].ID

	item := get[api.Item](t, h, api.Prefix+"/contents/"+id, http.StatusOK)
	if item.ID != id {
		t.Errorf("id = %q, want %q", item.ID, id)
	}
	if !strings.Contains(item.Body, "marmalade") {
		t.Errorf("body does not carry the source: %q", item.Body)
	}
	if item.BodyFormat != "markdown" {
		t.Errorf("body_format = %q, want markdown", item.BodyFormat)
	}

	// A listing row must not carry the body: it is the difference between a
	// list request that stays small and one that grows with the corpus.
	var raw map[string]any
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, api.Prefix+"/contents?kind=post", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	first := raw["items"].([]any)[0].(map[string]any)
	if _, present := first["body"]; present {
		t.Error("a listing row carries the full body")
	}
}

// A known endpoint asked for content that is not there answers about the
// content. Only an unknown path answers about the endpoint: telling an author
// their id is wrong when the endpoint is missing, or the reverse, sends them
// looking in the wrong place.
func TestMissingItemIsAboutTheItemNotTheEndpoint(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	missing := get[api.ErrorBody](t, h, api.Prefix+"/contents/01J000000000000000000000", http.StatusNotFound)
	if missing.Error.Code != api.CodeNotFound {
		t.Errorf("code = %q, want %q", missing.Error.Code, api.CodeNotFound)
	}
	if strings.Contains(missing.Error.Message, "endpoint") {
		t.Errorf("message talks about the endpoint, not the content: %q", missing.Error.Message)
	}
	if !strings.Contains(missing.Error.Message, "01J000000000000000000000") {
		t.Errorf("message does not name the id asked for: %q", missing.Error.Message)
	}

	unknown := get[api.ErrorBody](t, h, api.Prefix+"/nope", http.StatusNotFound)
	if !strings.Contains(unknown.Error.Message, "endpoint") {
		t.Errorf("unknown path does not say the endpoint is unknown: %q", unknown.Error.Message)
	}
}

// A read-only endpoint is refused by the router, not by a handler that
// remembered to check.
func TestWritesAreRefused(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, api.Prefix+"/contents", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /contents: status = %d, want 405", method, rec.Code)
		}
	}
}

// An unknown path under the API prefix must answer in the API's own language.
// Falling through to the site's HTML 404 would hand a client a page to parse.
func TestUnknownEndpointAnswersInJSON(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	body := get[api.ErrorBody](t, h, api.Prefix+"/nope", http.StatusNotFound)
	if body.Error.Code != api.CodeNotFound {
		t.Errorf("code = %q, want %q", body.Error.Code, api.CodeNotFound)
	}
}

func TestContentTypesCarryTheFieldSchemaFormsAreBuiltFrom(t *testing.T) {
	h, _ := newServer(t, newProject(t, 1))

	list := get[api.List[api.ContentType]](t, h, api.Prefix+"/content-types", http.StatusOK)
	if len(list.Items) == 0 {
		t.Fatal("no content types")
	}

	var post *api.ContentType
	for i, ct := range list.Items {
		if ct.Kind == "post" {
			post = &list.Items[i]
		}
	}
	if post == nil {
		t.Fatal("no post type")
	}
	if len(post.Fields) == 0 {
		t.Error("post type carries no field schema, so an admin form cannot be generated from it")
	}
	if post.Route == "" || post.Layout == "" {
		t.Errorf("route = %q, layout = %q, want both", post.Route, post.Layout)
	}
}

func TestTaxonomyTermsAreCountedAndLinked(t *testing.T) {
	h, _ := newServer(t, newProject(t, 10))

	taxonomies := get[api.List[api.Taxonomy]](t, h, api.Prefix+"/taxonomies", http.StatusOK)
	names := map[string]int{}
	for _, tx := range taxonomies.Items {
		names[tx.Name] = tx.Terms
	}
	if names["tags"] != 2 {
		t.Errorf("tags has %d terms, want 2", names["tags"])
	}

	terms := get[api.List[api.TermCount]](t, h, api.Prefix+"/taxonomies/tags/terms", http.StatusOK)
	got := map[string]int{}
	for _, tc := range terms.Items {
		got[tc.Term] = tc.Count
		if tc.URL == "" {
			t.Errorf("term %q has no URL", tc.Term)
		}
	}
	if got["Go"] != 10 || got["Notes"] != 5 {
		t.Errorf("term counts = %v, want Go:10 Notes:5", got)
	}

	// An aggregate over a filtered set is the whole point of taking a query.
	drafts := get[api.List[api.TermCount]](t, h, api.Prefix+"/taxonomies/tags/terms?status=draft", http.StatusOK)
	for _, tc := range drafts.Items {
		if tc.Term == "Go" && tc.Count != 2 {
			t.Errorf("draft Go count = %d, want 2", tc.Count)
		}
	}

	get[api.ErrorBody](t, h, api.Prefix+"/taxonomies/nonsense/terms", http.StatusNotFound)
}

func TestSiteReportsWhatTheIndexRefused(t *testing.T) {
	root := newProject(t, 3)
	// A file with no id is reported, never repaired: only its author knows
	// what it was meant to say. The admin has to be able to show it, or the
	// file just silently is not there.
	write(t, filepath.Join(root, "content", "posts", "orphan", "index.md"),
		"---\ntitle: No id\n---\n\nbody\n")

	h, _ := newServer(t, root)

	info := get[api.SiteInfo](t, h, api.Prefix+"/site", http.StatusOK)
	if len(info.Problems) == 0 {
		t.Fatal("site reports no problems although one file has no id")
	}
	if !strings.Contains(strings.Join(info.Problems, "\n"), "orphan") {
		t.Errorf("problems do not name the file: %v", info.Problems)
	}
	if info.Title != "Field Notes" {
		t.Errorf("title = %q", info.Title)
	}
	if info.Counts["post"] != 3 {
		t.Errorf("post count = %d, want 3 (the unreadable file is not one)", info.Counts["post"])
	}
	if info.Store != "file" {
		t.Errorf("store = %q, want file", info.Store)
	}
}

// The admin is a view of the files, not a second source of truth: an edit
// made in an editor has to show up without the admin being told.
func TestListReflectsAnEditMadeOutsideTheAdmin(t *testing.T) {
	root := newProject(t, 3)
	h, s := newServer(t, root)

	before := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&count=true", http.StatusOK)
	if *before.Total != 3 {
		t.Fatalf("total = %d, want 3", *before.Total)
	}

	write(t, filepath.Join(root, "content", "posts", "post-99", "index.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ999\ntitle: Written in an editor\nslug: post-99\n"+
			"status: published\npublished_at: 2026-02-01T00:00:00Z\ntags: [Go]\n---\n\nbody\n")
	if _, err := s.Index.Reconcile(t.Context()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	after := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&count=true", http.StatusOK)
	if *after.Total != 4 {
		t.Errorf("total = %d, want 4 after a file appeared", *after.Total)
	}
	if after.Items[0].Slug != "post-99" {
		t.Errorf("newest = %q, want post-99", after.Items[0].Slug)
	}
}
