package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/site"
)

// send issues a request with a body and returns the recorder, so a test can
// read the status, the headers and the body together.
func send(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(v))
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, rec.Body.String())
	}
	return v
}

// load reads an item and the entity tag that goes back in If-Match.
func load(t *testing.T, h http.Handler, id string) (api.Item, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, api.Prefix+"/contents/"+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: %d\n%s", id, rec.Code, rec.Body.String())
	}
	tag := rec.Header().Get("ETag")
	if tag == "" {
		t.Fatal("a read returned no ETag, so an edit has no revision to send back")
	}
	return decode[api.Item](t, rec), tag
}

func draftOf(item api.Item) api.Draft {
	return api.Draft{
		Kind:        item.Kind,
		Title:       item.Title,
		Slug:        item.Slug,
		Status:      item.Status,
		Body:        item.Body,
		Meta:        item.Meta,
		Taxonomies:  item.Taxonomies,
		Aliases:     item.Aliases,
		Locale:      item.Locale,
		PublishedAt: item.PublishedAt,
	}
}

func TestCreateStoresAnItemAndReportsWhereItLives(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	rec := send(t, h, http.MethodPost, api.Prefix+"/contents", api.Draft{
		Kind:   "post",
		Title:  "Written in the admin",
		Status: "draft",
		Body:   "Some prose.\n",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201\n%s", rec.Code, rec.Body.String())
	}

	created := decode[api.Item](t, rec)
	if created.ID == "" {
		t.Fatal("no id was assigned")
	}
	if created.Slug != "written-in-the-admin" {
		t.Errorf("slug = %q, want one derived from the title", created.Slug)
	}
	if rec.Header().Get("ETag") == "" {
		t.Error("no ETag, so the client cannot immediately edit what it just made")
	}
	if got, want := rec.Header().Get("Location"), api.Prefix+"/contents/"+created.ID; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}

	// The bytes have to be on disk, not merely in the index.
	source := filepath.Join(root, filepath.FromSlash(created.Locator), "index.md")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("no file was written: %v", err)
	}
	if !strings.Contains(string(data), "Some prose.") {
		t.Errorf("the file does not carry the body:\n%s", data)
	}
}

// This is the M3 promise that decides whether anyone will point this at a
// repository they care about: editing one field changes one field.
func TestEditingTheTitleRewritesOnlyTheTitleLine(t *testing.T) {
	root := newProject(t, 3)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)

	source := filepath.Join(root, filepath.FromSlash(item.Locator), "index.md")
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	draft := draftOf(item)
	draft.Title = "A brand new title"
	rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body.String())
	}

	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	changed := changedLines(string(before), string(after))
	if len(changed) != 1 {
		t.Errorf("editing the title changed %d lines, want 1:\n%s", len(changed), strings.Join(changed, "\n"))
	}
	if len(changed) > 0 && !strings.Contains(changed[0], "A brand new title") {
		t.Errorf("the changed line is not the title: %q", changed[0])
	}
}

// changedLines reports the lines that differ, by position, between two files
// of the same length, or every line when the length changed.
func changedLines(before, after string) []string {
	b, a := strings.Split(before, "\n"), strings.Split(after, "\n")
	if len(b) != len(a) {
		return append([]string{"line count changed"}, a...)
	}
	var out []string
	for i := range a {
		if a[i] != b[i] {
			out = append(out, "-"+b[i]+"\n+"+a[i])
		}
	}
	return out
}

// An edit made against a version that has since been replaced must not win by
// arriving second.
func TestSavingAgainstAReplacedVersionIsRefusedWithWhatIsStored(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, stale := load(t, h, list.Items[0].ID)

	// Somebody else saves first.
	first := draftOf(item)
	first.Body = "Written by the other editor.\n"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, first,
		map[string]string{"If-Match": stale}); rec.Code != http.StatusOK {
		t.Fatalf("the first save failed: %d\n%s", rec.Code, rec.Body.String())
	}

	// Now the stale editor saves, still holding the old revision.
	second := draftOf(item)
	second.Body = "Written by the editor that had gone stale.\n"
	rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, second,
		map[string]string{"If-Match": stale})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	body := decode[api.ConflictBody](t, rec)
	if body.Error.Code != api.CodeConflict {
		t.Errorf("code = %q, want %q", body.Error.Code, api.CodeConflict)
	}
	if body.Conflict.Theirs == nil {
		t.Fatal("the conflict carries no stored version, so the client has nothing to compare against")
	}
	if !strings.Contains(body.Conflict.Theirs.Body, "the other editor") {
		t.Errorf("theirs is not what is stored: %q", body.Conflict.Theirs.Body)
	}
	if body.Conflict.ActualRevision == body.Conflict.ExpectedRevision {
		t.Error("the two revisions are reported as equal, which is not a conflict")
	}
	// The losing edit must not have reached the file.
	if strings.Contains(body.Conflict.Theirs.Body, "had gone stale") {
		t.Error("the stale edit overwrote the newer one")
	}
	if rec.Header().Get("ETag") == "" {
		t.Error("no ETag on the conflict, so the client cannot retry without re-reading")
	}
}

// Forgetting the header must not mean "overwrite whatever is there".
func TestASaveWithoutAPreconditionIsRefused(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, _ := load(t, h, list.Items[0].ID)

	for _, tc := range []struct {
		name   string
		header map[string]string
		want   int
	}{
		{"absent", nil, http.StatusPreconditionRequired},
		{"wildcard", map[string]string{"If-Match": "*"}, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draftOf(item), tc.header)
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d\n%s", rec.Code, tc.want, rec.Body.String())
			}
			if field := decode[api.ErrorBody](t, rec).Error.Field; field != "If-Match" {
				t.Errorf("field = %q, want If-Match", field)
			}
		})
	}
}

// A field the server owns must be refused, not quietly dropped: a client that
// sends one believes it is being honored.
func TestSendingAServerOwnedFieldIsRefused(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	for _, body := range []string{
		`{"kind":"post","title":"x","status":"draft","body":"","id":"01J8KQ2P3R4S5T6V7W8X9YZAB1"}`,
		`{"kind":"post","title":"x","status":"draft","body":"","revision":"abc"}`,
		`{"kind":"post","title":"x","status":"draft","body":"","locator":"content/posts/x"}`,
		`{"kind":"post","title":"x","status":"draft","body":"","updated_at":"2026-01-01T00:00:00Z"}`,
	} {
		rec := send(t, h, http.MethodPost, api.Prefix+"/contents", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 for %s", rec.Code, body)
		}
	}
}

func TestInvalidContentIsAClientError(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	for _, tc := range []struct{ name, body string }{
		{"unknown kind", `{"kind":"nonsense","title":"x","status":"draft","body":""}`},
		{"unknown status", `{"kind":"post","title":"x","status":"nonsense","body":""}`},
		{"no kind", `{"title":"x","status":"draft","body":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPost, api.Prefix+"/contents", tc.body, nil)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400\n%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestDeleteRemovesTheBundleAndRefusesAStalePrecondition(t *testing.T) {
	root := newProject(t, 3)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	dir := filepath.Join(root, filepath.FromSlash(item.Locator))

	if rec := send(t, h, http.MethodDelete, api.Prefix+"/contents/"+item.ID, nil,
		map[string]string{"If-Match": `"not-the-revision"`}); rec.Code != http.StatusConflict {
		t.Fatalf("a stale delete returned %d, want 409", rec.Code)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("the refused delete removed the bundle anyway")
	}

	if rec := send(t, h, http.MethodDelete, api.Prefix+"/contents/"+item.ID, nil,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusNoContent {
		t.Fatalf("delete returned %d, want 204", rec.Code)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("the bundle is still on disk")
	}
	get[api.ErrorBody](t, h, api.Prefix+"/contents/"+item.ID, http.StatusNotFound)
}

// A write has to be visible to the next read without waiting for the file
// watcher, or a UI learns to keep its own copy of the truth.
func TestAWriteIsVisibleToTheVeryNextRead(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	before := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&count=true", http.StatusOK)

	rec := send(t, h, http.MethodPost, api.Prefix+"/contents", api.Draft{
		Kind: "post", Title: "Just now", Status: "published", Body: "x",
	}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d\n%s", rec.Code, rec.Body.String())
	}

	after := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&count=true", http.StatusOK)
	if *after.Total != *before.Total+1 {
		t.Errorf("total went %d -> %d, want one more", *before.Total, *after.Total)
	}
}

// Reading a repository over HTTP and letting anyone who reaches the port
// rewrite it are not the same decision.
func TestAReadOnlyServerRefusesEveryWrite(t *testing.T) {
	h, _ := newServer(t, newProject(t, 2))

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	id := list.Items[0].ID

	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, api.Prefix + "/contents", api.Draft{Kind: "post", Title: "x", Status: "draft"}},
		{http.MethodPut, api.Prefix + "/contents/" + id, api.Draft{Kind: "post", Title: "x", Status: "draft"}},
		{http.MethodDelete, api.Prefix + "/contents/" + id, nil},
	} {
		rec := send(t, h, tc.method, tc.path, tc.body, map[string]string{"If-Match": `"whatever"`})
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want 405", tc.method, tc.path, rec.Code)
		}
		if code := decode[api.ErrorBody](t, rec).Error.Code; code != api.CodeReadOnly {
			t.Errorf("%s %s: code = %q, want %q", tc.method, tc.path, code, api.CodeReadOnly)
		}
	}
}

// newWritableServer is newServer with the project open for writing.
func newWritableServer(t *testing.T, root string) (http.Handler, *site.Site) {
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
			Writer:   s.Project.Writer(),
			Refresh: func(ctx context.Context) error {
				_, err := s.Index.Reconcile(ctx)
				return err
			},
		}
	}})

	mux := http.NewServeMux()
	srv.Mount(mux)
	return mux, s
}

// The other half of the timestamp rule: a value the author does keep must go
// on being maintained, or "Kite does not touch my files" would mean "Kite
// does not do its job".
func TestATimestampTheAuthorKeepsIsMaintained(t *testing.T) {
	root := newProject(t, 1)
	write(t, filepath.Join(root, "content", "posts", "tracked", "index.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ700\ntitle: Tracked\nslug: tracked\n"+
			"status: published\nupdated_at: 2020-01-01T00:00:00Z\n---\n\nbody\n")

	h, _ := newWritableServer(t, root)
	item, tag := load(t, h, "01J8KQ2P3R4S5T6V7W8X9YZ700")

	draft := draftOf(item)
	draft.Body = "changed\n"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}

	data, err := os.ReadFile(filepath.Join(root, "content", "posts", "tracked", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "2020-01-01") {
		t.Error("updated_at was declared by the author but not maintained")
	}
	if !strings.Contains(string(data), "updated_at:") {
		t.Error("updated_at was removed")
	}
	// Nothing the author did not keep may appear.
	if strings.Contains(string(data), "created_at:") {
		t.Error("created_at was invented on a file that never had one")
	}
}
