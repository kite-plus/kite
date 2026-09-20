package serve_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/serve"
	"github.com/kite-plus/kite/internal/site"
)

// frozen keeps the clock out of the comparison. It is an input like any
// other, and the whole design says an input has to be explicit.
var frozen = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

const config = `site:
  title: Parity
  description: Build and serve must agree.
  baseURL: https://example.com
  language: en
build:
  output: public
  pageSize: 3
`

func post(i int, extra string) string {
	return fmt.Sprintf(`---
id: 01J8KQ2P3R4S5T6V7W8X9YZ%03d
title: Post %02d
slug: post-%02d
status: published
published_at: 2026-01-%02dT00:00:00Z
%s---

# Heading %02d

Body with **bold**, a [link](https://example.com) and `+"`code`"+`.

## Second heading

`+"```go"+`
func main() {}
`+"```"+`
`, i, i, i, i+1, extra, i)
}

func newProject(t *testing.T, posts int) string {
	t.Helper()
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "kite.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := range posts {
		tags := "tags: [Go]\n"
		if i%2 == 0 {
			tags = "tags: [Go, Notes]\ncategories: [Tech]\n"
		}
		dir := filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(post(i, tags)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pages := filepath.Join(root, "content", "pages")
	if err := os.MkdirAll(pages, 0o755); err != nil {
		t.Fatal(err)
	}
	about := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ900\ntitle: About\nslug: about\nstatus: published\n---\n\nAbout this site.\n"
	if err := os.WriteFile(filepath.Join(pages, "about.md"), []byte(about), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func openSite(t *testing.T, root string) *site.Site {
	t.Helper()
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatalf("site.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// This is the claim the whole two-axes design rests on: where content lives
// and how it is delivered are independent, so the same content delivered two
// ways is the same bytes. If this test ever fails, a theme author can no
// longer trust the preview, and the claim is marketing.
func TestServedPagesAreByteIdenticalToBuiltFiles(t *testing.T) {
	root := newProject(t, 7)

	built := openSite(t, root)
	outDir := filepath.Join(root, "public")
	if _, files, err := built.Build(t.Context(), site.BuildOptions{OutDir: outDir, Now: frozen}); err != nil {
		t.Fatalf("Build: %v", err)
	} else if len(files) == 0 {
		t.Fatal("build produced nothing to compare against")
	}

	srv := newServer(t, root, serve.Options{LiveReload: false})
	handler := srv.Handler()

	var checked int
	err := filepath.WalkDir(outDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".html" {
			return err
		}
		rel, err := filepath.Rel(outDir, p)
		if err != nil {
			return err
		}
		url := "/" + filepath.ToSlash(rel)

		want, err := os.ReadFile(p)
		if err != nil {
			return err
		}

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))

		// 404.html is served with its own status, everything else with 200.
		if wantStatus := statusFor(rel); rec.Code != wantStatus {
			t.Errorf("%s: status %d, want %d", url, rec.Code, wantStatus)
			return nil
		}
		if got := rec.Body.String(); got != string(want) {
			t.Errorf("%s differs between build and serve\n%s", url, firstDifference(string(want), got))
			return nil
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 10 {
		t.Fatalf("only %d pages compared; the fixture is not exercising enough of the site", checked)
	}
	t.Logf("compared %d pages byte for byte", checked)
}

func statusFor(rel string) int {
	if rel == "404.html" {
		return http.StatusNotFound
	}
	return http.StatusOK
}

func newServer(t *testing.T, root string, opts serve.Options) *serve.Server {
	t.Helper()
	s := openSite(t, root)
	// The server freezes its own clock per reload; pinning it here is what
	// lets the comparison be about rendering rather than about time.
	srv, err := serve.NewWithClock(context.Background(), s, opts, func() time.Time { return frozen })
	if err != nil {
		t.Fatalf("serve.New: %v", err)
	}
	return srv
}

func TestURLSpellingsResolveToOnePage(t *testing.T) {
	root := newProject(t, 3)
	handler := newServer(t, root, serve.Options{}).Handler()

	bodies := map[string]string{}
	for _, url := range []string{
		"/posts/post-00/",
		"/posts/post-00",
		"/posts/post-00/index.html",
		"/posts/post-00/?draft=1",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", url, rec.Code)
			continue
		}
		bodies[url] = rec.Body.String()
	}
	for url, body := range bodies {
		if body != bodies["/posts/post-00/"] {
			t.Errorf("%s served a different page than the canonical URL", url)
		}
	}
}

func TestUnknownPathServesTheNotFoundPage(t *testing.T) {
	root := newProject(t, 2)
	handler := newServer(t, root, serve.Options{}).Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope/", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Not found") {
		t.Errorf("the 404 template was not used:\n%s", rec.Body.String())
	}
}

// The reload script is the one thing a served page carries that a built one
// does not, so it has to be switchable and has to land inside the document.
func TestLiveReloadInjection(t *testing.T) {
	root := newProject(t, 2)

	off := newServer(t, root, serve.Options{LiveReload: false}).Handler()
	rec := httptest.NewRecorder()
	off.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if strings.Contains(rec.Body.String(), serve.ReloadPath) {
		t.Error("a reload script was injected with live reload off")
	}

	on := newServer(t, root, serve.Options{LiveReload: true}).Handler()
	rec = httptest.NewRecorder()
	on.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, serve.ReloadPath) {
		t.Error("no reload script was injected with live reload on")
	}
	if strings.Index(body, serve.ReloadPath) > strings.LastIndex(body, "</body>") {
		t.Error("the reload script was placed outside the document body")
	}
	if got := rec.Result().Header.Get("Content-Length"); got != fmt.Sprint(len(body)) {
		t.Errorf("Content-Length = %s, want %d after injection", got, len(body))
	}
}

func TestReloadEndpointOnlyExistsWhenEnabled(t *testing.T) {
	root := newProject(t, 1)

	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{LiveReload: false}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, serve.ReloadPath, nil))
	if rec.Code == http.StatusOK {
		t.Error("the reload endpoint answered with live reload off")
	}
}

func TestStaticFilesAreServed(t *testing.T) {
	root := newProject(t, 1)
	if err := os.MkdirAll(filepath.Join(root, "static", "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "static", "img", "x.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/img/x.svg", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "<svg/>" {
		t.Errorf("body = %q", rec.Body.String())
	}
	if ct := rec.Result().Header.Get("Content-Type"); !strings.Contains(ct, "svg") {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestStaticPathCannotEscapeTheProject(t *testing.T) {
	root := newProject(t, 1)
	for _, url := range []string{"/../kite.yaml", "/..%2fkite.yaml", "/static/../../etc/hosts"} {
		rec := httptest.NewRecorder()
		newServer(t, root, serve.Options{}).Handler().
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "baseURL") {
			t.Errorf("%s escaped the project root", url)
		}
	}
}

func TestDraftsAreHiddenUnlessRequested(t *testing.T) {
	root := newProject(t, 2)
	dir := filepath.Join(root, "content", "posts", "hidden")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ950\ntitle: Hidden\nslug: hidden\nstatus: draft\n---\n\nsecret\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/hidden/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a draft was served without --drafts (status %d)", rec.Code)
	}

	rec = httptest.NewRecorder()
	newServer(t, root, serve.Options{Drafts: true}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/hidden/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("--drafts did not reveal the draft (status %d)", rec.Code)
	}
}

func TestEditedContentIsServedAfterReload(t *testing.T) {
	root := newProject(t, 2)
	srv := newServer(t, root, serve.Options{})
	handler := srv.Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/post-00/", nil))
	if !strings.Contains(rec.Body.String(), "Post 00") {
		t.Fatalf("unexpected initial body:\n%s", rec.Body.String())
	}

	p := filepath.Join(root, "content", "posts", "post-00", "index.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "title: Post 00", "title: Renamed", 1)
	if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := srv.Reload(t.Context()); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/post-00/", nil))
	if !strings.Contains(rec.Body.String(), "Renamed") {
		t.Error("the edit was not picked up after a reload")
	}
}

func TestHeadRequestSendsNoBody(t *testing.T) {
	root := newProject(t, 1)
	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD returned %d bytes of body", rec.Body.Len())
	}
	if rec.Result().Header.Get("Content-Length") == "" {
		t.Error("HEAD should still report the length")
	}
}

// firstDifference points at where two renderings diverge, because a diff of
// two full pages is unreadable in test output.
func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range max(len(w), len(g)) {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return fmt.Sprintf("  line %d\n   built:  %q\n   served: %q", i+1, a, b)
		}
	}
	return "  (identical line by line but different as bytes)"
}

// A file the index refuses is a file that silently does not appear. The
// terminal warning is not where the author is looking, so the preview has to
// say so on the page.
func TestUnindexableFileIsReportedInThePage(t *testing.T) {
	root := newProject(t, 2)
	dir := filepath.Join(root, "content", "posts", "no-id")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: Created by hand\n---\n\nNo id yet.\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{LiveReload: true}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	page := rec.Body.String()
	if !strings.Contains(page, "not indexed") {
		t.Error("the page gives no sign that a file was skipped")
	}
	if !strings.Contains(page, "no-id/index.md") {
		t.Error("the banner does not name the file")
	}
	if !strings.Contains(page, "fix-ids") {
		t.Error("the banner does not say what to do about it")
	}
}

func TestNoBannerWhenEverythingIndexes(t *testing.T) {
	root := newProject(t, 2)
	rec := httptest.NewRecorder()
	newServer(t, root, serve.Options{LiveReload: true}).Handler().
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if strings.Contains(rec.Body.String(), "not indexed") {
		t.Error("a banner appeared with nothing to report")
	}
}
