package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/site"
	"github.com/kite-plus/kite/internal/store/file"
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

// Posting the same draft twice is an ordinary thing to do, and both items have
// to come back readable. The second used to be written to a folder of its own
// while keeping the first one's slug, so the index refused the row and the
// create answered 404 for the item it had just written.
func TestCreatingTwoItemsWithOneTitleKeepsBothReadable(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)

	draft := api.Draft{Kind: "post", Title: "tmp roundtrip", Status: "draft", Body: "hi"}
	first := send(t, h, http.MethodPost, api.Prefix+"/contents", draft, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("first create: status = %d\n%s", first.Code, first.Body.String())
	}
	second := send(t, h, http.MethodPost, api.Prefix+"/contents", draft, nil)
	if second.Code != http.StatusCreated {
		t.Fatalf("second create: status = %d\n%s", second.Code, second.Body.String())
	}

	one, two := decode[api.Item](t, first), decode[api.Item](t, second)
	if one.Slug != "tmp-roundtrip" || two.Slug != "tmp-roundtrip-2" {
		t.Errorf("slugs = %q and %q, want tmp-roundtrip and tmp-roundtrip-2", one.Slug, two.Slug)
	}
	// Folder and slug have to agree, or the URL and the bytes part company.
	if two.Locator != "content/posts/tmp-roundtrip-2" {
		t.Errorf("locator = %q, want the folder the slug names", two.Locator)
	}
	for _, id := range []string{one.ID, two.ID} {
		load(t, h, id)
	}
}

// A slug the author chose is refused rather than renamed, and the refusal
// comes before the file is written: an orphan on disk would be reported as
// missing and would block every later reconcile.
func TestCreatingWithATakenSlugIsRefusedAndLeavesNoFile(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	rec := send(t, h, http.MethodPost, api.Prefix+"/contents", api.Draft{
		Kind: "post", Title: "Another", Slug: "post-00", Status: "draft", Body: "hi",
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\n%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "post-00") {
		t.Errorf("the error should name the slug it refused: %s", body)
	}

	entries, err := os.ReadDir(filepath.Join(root, "content", "posts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("%d bundles on disk, want the original 2", len(entries))
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

func TestDeleteKeepsTheBundleRecoverableAndRefusesAStalePrecondition(t *testing.T) {
	root := newProject(t, 3)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	dir := filepath.Join(root, filepath.FromSlash(item.Locator))
	write(t, filepath.Join(dir, "cover.webp"), "attachment")

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
	if _, err := os.Stat(filepath.Join(dir, "cover.webp")); err != nil {
		t.Errorf("soft delete lost the attachment: %v", err)
	}
	active := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post", http.StatusOK)
	for _, entry := range active.Items {
		if entry.ID == item.ID {
			t.Error("deleted item is still in the active list")
		}
	}
	trash := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&deleted_only=true", http.StatusOK)
	if len(trash.Items) != 1 || trash.Items[0].ID != item.ID {
		t.Fatalf("trash = %+v, want only %s", trash.Items, item.ID)
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/contents/"+item.ID+"/restore", nil,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusConflict {
		t.Fatalf("stale restore returned %d, want 409", rec.Code)
	}
	_, deletedTag := load(t, h, item.ID)
	if rec := send(t, h, http.MethodPost, api.Prefix+"/contents/"+item.ID+"/restore", nil,
		map[string]string{"If-Match": deletedTag}); rec.Code != http.StatusOK {
		t.Fatalf("restore returned %d, want 200\n%s", rec.Code, rec.Body.String())
	}
	trash = get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&deleted_only=true", http.StatusOK)
	if len(trash.Items) != 0 {
		t.Errorf("restored item remains in trash: %+v", trash.Items)
	}
	if _, err := os.Stat(filepath.Join(dir, "cover.webp")); err != nil {
		t.Errorf("restore lost the attachment: %v", err)
	}
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
func newWritableServer(t *testing.T, root string, with ...func(*api.Options)) (http.Handler, *site.Site) {
	t.Helper()
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatalf("site.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// The site is held behind a pointer the refresh can replace, the way a
	// running server holds it: a settings change rebuilds everything derived
	// from the configuration.
	current := s
	configBytes, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configRevision := file.RevisionOf(configBytes)

	opts := api.Options{Site: func() api.View {
		return api.View{
			Reader:         current.Reader,
			Resolver:       current.Resolver,
			Types:          current.Project.Types,
			Site:           current.Config.Site,
			Store:          current.Config.Content.Store,
			Runtime:        "test",
			Theme:          current.Config.Theme.Name,
			ThemeSchema:    current.Theme.Manifest.Settings,
			ThemeValues:    current.ThemeSettings(),
			ConfigRevision: configRevision,
			Problems:       current.Problems,
			Writer:         current.Project.Writer(),
			Publisher:      current.Publisher(),
			Refresh: func(ctx context.Context) error {
				next, err := current.Reconfigure()
				if err != nil {
					return err
				}
				current = next
				configBytes, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
				if err != nil {
					return err
				}
				configRevision = file.RevisionOf(configBytes)
				_, err = current.Index.Reconcile(ctx)
				return err
			},
		}
	}}
	for _, apply := range with {
		apply(&opts)
	}
	srv := api.New(opts)

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

// upload posts a file the way a browser does when something is dropped on the
// editor.
func upload(t *testing.T, h http.Handler, id, name string, data []byte) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, api.Prefix+"/contents/"+id+"/media", &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDroppedFilesLandInTheBundleAndReportALinkTheMarkdownCanUse(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, _ := load(t, h, list.Items[0].ID)

	rec := upload(t, h, item.ID, "diagram.png", []byte("pretend png"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201\n%s", rec.Code, rec.Body.String())
	}
	media := decode[api.Media](t, rec)

	// The link is a bare name: a bundle publishes its files beside the page,
	// which is what keeps the markdown readable in an editor and on GitHub.
	if media.Link != "diagram.png" {
		t.Errorf("link = %q, want a bundle relative name", media.Link)
	}
	if media.URL != item.URL+"diagram.png" {
		t.Errorf("url = %q, want it under %q", media.URL, item.URL)
	}
	if media.Type != "image/png" {
		t.Errorf("type = %q, want image/png", media.Type)
	}

	stored := filepath.Join(root, filepath.FromSlash(media.Path))
	data, err := os.ReadFile(stored)
	if err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	if string(data) != "pretend png" {
		t.Errorf("stored bytes = %q", data)
	}
	if got, want := filepath.Dir(stored), filepath.Join(root, filepath.FromSlash(item.Locator)); got != want {
		t.Errorf("stored in %s, want the item's own bundle %s", got, want)
	}
}

// Two screenshots are both called screenshot.png. Losing one of them is not a
// reasonable reading of "put this here".
func TestASecondFileOfTheSameNameDoesNotReplaceTheFirst(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, _ := load(t, h, list.Items[0].ID)

	first := decode[api.Media](t, upload(t, h, item.ID, "shot.png", []byte("the first one")))
	second := decode[api.Media](t, upload(t, h, item.ID, "shot.png", []byte("the second one")))

	if first.Name == second.Name {
		t.Fatalf("both uploads were stored as %q, so one overwrote the other", first.Name)
	}
	for _, m := range []struct {
		media api.Media
		want  string
	}{{first, "the first one"}, {second, "the second one"}} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(m.media.Path)))
		if err != nil {
			t.Fatalf("%s is gone: %v", m.media.Name, err)
		}
		if string(data) != m.want {
			t.Errorf("%s holds %q, want %q", m.media.Name, data, m.want)
		}
	}
}

// A bundle is published from the site's own origin, so what may be stored
// there is a list, not whatever was dropped.
func TestAFileTypeABundleWillNotHoldIsRefused(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	id := list.Items[0].ID

	for _, name := range []string{"payload.html", "script.js", "run.sh", "noextension"} {
		rec := upload(t, h, id, name, []byte("x"))
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("%s: status = %d, want 415", name, rec.Code)
		}
	}
}

// A name that tries to climb out of the bundle must land inside it anyway.
func TestAnUploadCannotEscapeTheBundle(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, _ := load(t, h, list.Items[0].ID)

	rec := upload(t, h, item.ID, "../../../escaped.png", []byte("x"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}
	media := decode[api.Media](t, rec)

	if !strings.HasPrefix(media.Path, string(item.Locator)+"/") {
		t.Errorf("stored at %q, which is outside the bundle %q", media.Path, item.Locator)
	}
	if _, err := os.Stat(filepath.Join(root, "escaped.png")); err == nil {
		t.Error("a file was written outside the bundle")
	}
}

func TestRemovingAFileTakesItOffDisk(t *testing.T) {
	root := newProject(t, 2)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, _ := load(t, h, list.Items[0].ID)

	media := decode[api.Media](t, upload(t, h, item.ID, "gone.png", []byte("x")))
	stored := filepath.Join(root, filepath.FromSlash(media.Path))

	rec := send(t, h, http.MethodDelete,
		api.Prefix+"/contents/"+item.ID+"/media/"+media.Name, nil, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204\n%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(stored); err == nil {
		t.Error("the file is still on disk")
	}
}

// A settings change goes through a change set like everything else, so the
// publisher will stage it the same way it stages a post, and so it lands in
// the file without disturbing what is around it.
func TestChangingASettingLeavesTheRestOfTheFileAlone(t *testing.T) {
	root := newProject(t, 1)
	config := filepath.Join(root, "kite.yaml")
	write(t, config, `# What visitors see.
site:
  title: Field Notes     # shown in the header
  description: A test project.
  baseURL: https://example.com
  language: en

# Rendering.
build:
  pageSize: 10
`)

	h, _ := newWritableServer(t, root)
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	rec := send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{
		"site.title":     "Renamed In The Admin",
		"build.pageSize": 25,
	}, map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body.String())
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	for _, want := range []string{
		"title: Renamed In The Admin",
		"pageSize: 25",
		"# What visitors see.",
		"# shown in the header",
		"# Rendering.",
		"description: A test project.",
		"language: en",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the file lost or never gained %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "site:") > strings.Index(got, "build:") {
		t.Errorf("the sections were reordered:\n%s", got)
	}

	// The response reports what is now stored.
	settings := decode[api.Settings](t, rec)
	if settings.Site.Title != "Renamed In The Admin" {
		t.Errorf("the response says %q", settings.Site.Title)
	}
}

func TestSettingsRequireCurrentConfigurationRevision(t *testing.T) {
	root := newProject(t, 1)
	configPath := filepath.Join(root, "kite.yaml")
	h, _ := newWritableServer(t, root)
	read := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil)
	oldTag := read.Header().Get("ETag")
	if oldTag == "" {
		t.Fatal("GET /settings returned no ETag")
	}

	missing := send(t, h, http.MethodPut, api.Prefix+"/settings",
		map[string]any{"site.title": "Browser edit"}, nil)
	if missing.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match: status = %d, want 428", missing.Code)
	}

	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(before), "Field Notes", "External edit", 1)
	if updated == string(before) {
		t.Fatal("fixture has no site title to change")
	}
	write(t, configPath, updated)

	stale := send(t, h, http.MethodPut, api.Prefix+"/settings",
		map[string]any{"site.title": "Browser edit"}, map[string]string{"If-Match": oldTag})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale edit: status = %d, want 409\n%s", stale.Code, stale.Body.String())
	}
	if decode[api.ErrorBody](t, stale).Error.Code != api.CodeConflict {
		t.Error("stale edit did not return the conflict code")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != updated {
		t.Fatalf("stale edit overwrote the external change:\n%s", after)
	}
	if stale.Header().Get("ETag") == oldTag {
		t.Error("conflict returned the stale ETag")
	}
}

// The configuration file is the one place where a mistake breaks the whole
// site, so what may be written is a list rather than a rule.
func TestASettingNobodyDesignedAControlForIsRefused(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 1))
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	for _, path := range []string{
		"content.store",   // switching the store is not a form control
		"build.output",    // nor is where the site is written
		"site.title.evil", // nor is a path that is not a setting
	} {
		rec := send(t, h, http.MethodPut, api.Prefix+"/settings",
			map[string]any{path: "x"}, map[string]string{"If-Match": tag})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", path, rec.Code)
		}
		if field := decode[api.ErrorBody](t, rec).Error.Field; field != path {
			t.Errorf("%s: field = %q", path, field)
		}
	}
}

// A theme declares what it can be configured with, so a theme author gets a
// settings form without writing any admin code.
func TestSettingsCarryTheThemesOwnSchema(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 1))

	settings := get[api.Settings](t, h, api.Prefix+"/settings", http.StatusOK)
	if settings.Theme.Name == "" {
		t.Error("no theme is named")
	}
	if len(settings.Theme.Schema) == 0 {
		t.Error("the theme declares no settings, so no form can be generated from it")
	}
	if !slices.Contains(settings.Writable, "site.title") {
		t.Errorf("writable does not list site.title: %v", settings.Writable)
	}
}

// A language tag becomes the lang attribute on every page and the prefix in
// every localized URL, so a typo in it does not fail: it produces a site that
// claims to be written in a language that does not exist.
func TestASettingThatWouldBreakTheSiteIsRefusedBeforeItIsWritten(t *testing.T) {
	root := newProject(t, 1)
	config := filepath.Join(root, "kite.yaml")
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}

	h, _ := newWritableServer(t, root)
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	for _, tc := range []struct{ name, path, value string }{
		{"a mistyped language", "site.language", "engrish!!"},
		{"a language with spaces", "site.language", "en US"},
		{"an address that is not one", "site.baseURL", "example.com"},
		{"no address at all", "site.baseURL", ""},
		{"a site with no name", "site.title", "   "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPut, api.Prefix+"/settings",
				map[string]any{tc.path: tc.value}, map[string]string{"If-Match": tag})

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400\n%s", rec.Code, rec.Body.String())
			}
			body := decode[api.ErrorBody](t, rec)
			if body.Error.Field != tc.path {
				t.Errorf("field = %q, want %q", body.Error.Field, tc.path)
			}
			// Refused before writing: a value rejected on the reload that
			// follows a write would leave the file broken and the project
			// unable to open.
			after, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Errorf("the file was written anyway:\n%s", after)
			}
		})
	}
}

// Clearing the language is not the same as breaking it: a project that never
// named one has always worked, and an emptied form field should fall back
// rather than produce <html lang="">.
func TestClearingTheLanguageFallsBackToTheDefault(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	rec := send(t, h, http.MethodPut, api.Prefix+"/settings",
		map[string]any{"site.language": ""}, map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body.String())
	}

	settings := decode[api.Settings](t, rec)
	if settings.Site.Language != "en" {
		t.Errorf("language = %q, want the default", settings.Site.Language)
	}
}
