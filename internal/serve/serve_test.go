package serve_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/auth"
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
	return newProjectAt(t, posts, "https://example.com")
}

// newProjectAt is a project published at baseURL.
func newProjectAt(t *testing.T, posts int, baseURL string) string {
	t.Helper()
	root := t.TempDir()

	written := strings.Replace(config, "https://example.com", baseURL, 1)
	if err := os.WriteFile(filepath.Join(root, "kite.yaml"), []byte(written), 0o644); err != nil {
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
func TestServedFilesAreByteIdenticalToBuiltOnes(t *testing.T) {
	// A site published under a path, as a GitHub Pages project site is, is
	// served under that path, where its host would publish each file.
	for _, at := range []struct{ name, baseURL, prefix string }{
		{"root", "https://example.com", ""},
		{"path", "https://example.github.io/blog/", "/blog"},
	} {
		t.Run(at.name, func(t *testing.T) {
			root := newProjectAt(t, 7, at.baseURL)
			// A build writes the feed after the static files, over this one,
			// and a server has to answer with the same file.
			static := filepath.Join(root, "static")
			if err := os.MkdirAll(static, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(static, "rss.xml"), []byte("<rss>stale</rss>\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			built := openSite(t, root)
			outDir := filepath.Join(root, "public")
			if _, files, err := built.Build(t.Context(), site.BuildOptions{OutDir: outDir, Now: frozen}); err != nil {
				t.Fatalf("Build: %v", err)
			} else if len(files) == 0 {
				t.Fatal("build produced nothing to compare against")
			}

			srv := newServer(t, root, serve.Options{LiveReload: false})
			handler := srv.Handler()

			var checked []string
			err := filepath.WalkDir(outDir, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, err := filepath.Rel(outDir, p)
				if err != nil {
					return err
				}
				url := at.prefix + "/" + filepath.ToSlash(rel)

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
				checked = append(checked, url)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(checked) < 10 {
				t.Fatalf("only %d files compared; the fixture is not exercising enough of the site", len(checked))
			}
			// Hooks write these after the pages, and a server once had no
			// answer for them at all.
			for _, name := range []string{"rss.xml", "sitemap.xml"} {
				if url := at.prefix + "/" + name; !slices.Contains(checked, url) {
					t.Errorf("%s was not compared", url)
				}
			}
			t.Logf("compared %d files byte for byte", len(checked))
		})
	}
}

// A preview under the site's path shows the links the deployed site will
// have, so one that forgot the path breaks here rather than after a push.
func TestASiteUnderAPathIsServedUnderIt(t *testing.T) {
	root := newProjectAt(t, 3, "https://example.github.io/blog/")
	bundle := filepath.Join(root, "content", "posts", "post-00", "photo.png")
	if err := os.WriteFile(bundle, []byte("not really a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	static := filepath.Join(root, "static")
	if err := os.MkdirAll(static, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(static, "robots.txt"), []byte("User-agent: *\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := newServer(t, root, serve.Options{}).Handler()

	get := func(url string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		return rec
	}

	// The address the server prints leads to the site.
	if rec := get("/"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/blog/" {
		t.Errorf("/ answered %d to %q, want a redirect to /blog/", rec.Code, rec.Header().Get("Location"))
	}
	for _, url := range []string{
		"/blog/", "/blog", "/blog/posts/post-00/", "/blog/tags/go/",
		"/blog/rss.xml", "/blog/sitemap.xml",
		"/blog/posts/post-00/photo.png", "/blog/robots.txt",
	} {
		if rec := get(url); rec.Code != http.StatusOK {
			t.Errorf("%s: status %d, want 200", url, rec.Code)
		}
	}
	// What the host would not publish there, the preview does not serve.
	for _, url := range []string{
		"/posts/post-00/", "/rss.xml", "/posts/post-00/photo.png", "/robots.txt", "/blogger/",
	} {
		if rec := get(url); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", url, rec.Code)
		}
	}
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

// A server plans against a frozen clock, and nothing on disk changes when a
// post dated later falls due, so a server left running has to notice the
// time itself or go on hiding the post until some unrelated edit.
func TestAScheduledPostIsServedOnceItsTimeComes(t *testing.T) {
	// A published post dated later waits for its date just the same.
	for _, status := range []string{"scheduled", "published"} {
		t.Run(status, func(t *testing.T) {
			root := newProject(t, 2)
			dir := filepath.Join(root, "content", "posts", "launch-day")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			body := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ951\ntitle: Launch Day\nslug: launch-day\n" +
				"status: " + status + "\npublished_at: 2026-06-01T12:30:00Z\n---\n\nNot yet.\n"
			if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			var mu sync.Mutex
			now := frozen
			clock := func() time.Time {
				mu.Lock()
				defer mu.Unlock()
				return now
			}
			srv, err := serve.NewWithClock(t.Context(), openSite(t, root), serve.Options{}, clock)
			if err != nil {
				t.Fatal(err)
			}
			handler := srv.Handler()
			get := func(path string) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
				return rec
			}

			if rec := get("/posts/launch-day/"); rec.Code != http.StatusNotFound {
				t.Errorf("served before its time (status %d)", rec.Code)
			}
			if strings.Contains(get("/").Body.String(), "Launch Day") {
				t.Error("listed on the home page before its time")
			}

			mu.Lock()
			now = time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
			mu.Unlock()

			if rec := get("/posts/launch-day/"); rec.Code != http.StatusOK {
				t.Errorf("not served once its time came (status %d)", rec.Code)
			}
			if !strings.Contains(get("/").Body.String(), "Launch Day") {
				t.Error("not listed on the home page once its time came")
			}
		})
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

// The feed and the sitemap describe the site as the server plans it now: a
// post that falls due joins them, and so does an edit.
func TestTheFeedFollowsTheSite(t *testing.T) {
	root := newProject(t, 2)
	dir := filepath.Join(root, "content", "posts", "launch-day")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ951\ntitle: Launch Day\nslug: launch-day\n" +
		"status: published\npublished_at: 2026-06-01T12:30:00Z\n---\n\nNot yet.\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	now := frozen
	srv, err := serve.NewWithClock(t.Context(), openSite(t, root), serve.Options{}, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := srv.Handler()
	get := func(path string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
		return rec.Body.String()
	}

	if strings.Contains(get("/rss.xml"), "launch-day") || strings.Contains(get("/sitemap.xml"), "launch-day") {
		t.Error("the post is described before its time")
	}

	mu.Lock()
	now = time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	mu.Unlock()
	if !strings.Contains(get("/rss.xml"), "launch-day") || !strings.Contains(get("/sitemap.xml"), "launch-day") {
		t.Error("the post is not described once its time came")
	}

	p := filepath.Join(root, "content", "posts", "post-00", "index.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "title: Post 00", "title: A renamed post", 1)
	if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	// What the file watcher does on a change.
	if err := srv.Reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(get("/rss.xml"), "A renamed post") {
		t.Error("the feed still has the title from before the edit")
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

// A page bundle keeps an item's images beside its text so the markdown can
// link them relatively. Before this worked, a bundle image was a 404 in both
// runtimes while the page confidently pointed at it.
func TestAPageBundlesOwnFilesAreServedAndBuiltAlike(t *testing.T) {
	root := newProject(t, 3)

	dir := filepath.Join(root, "content", "posts", "post-01")
	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("pretend png"), 0o644); err != nil {
		t.Fatal(err)
	}
	appendTo(t, filepath.Join(dir, "index.md"), "\n![cover](cover.png)\n")

	built := openSite(t, root)
	outDir := filepath.Join(root, "public")
	if _, _, err := built.Build(t.Context(), site.BuildOptions{OutDir: outDir, Now: frozen}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	onDisk, err := os.ReadFile(filepath.Join(outDir, "posts", "post-01", "cover.png"))
	if err != nil {
		t.Fatalf("the build did not publish the bundle's image: %v", err)
	}

	srv := newServer(t, root, serve.Options{LiveReload: false})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/post-01/cover.png", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("serving the bundle's image returned %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), onDisk) {
		t.Error("the served image differs from the built one")
	}
	if ctype := rec.Header().Get("Content-Type"); !strings.HasPrefix(ctype, "image/png") {
		t.Errorf("content type = %q, want image/png", ctype)
	}

	// The markdown source is rendered, never published beside the page.
	if _, err := os.Stat(filepath.Join(outDir, "posts", "post-01", "index.md")); err == nil {
		t.Error("the markdown source was published alongside the page")
	}
	source := httptest.NewRecorder()
	srv.Handler().ServeHTTP(source, httptest.NewRequest(http.MethodGet, "/posts/post-01/index.md", nil))
	if source.Code == http.StatusOK {
		t.Error("the markdown source is reachable over HTTP")
	}
}

func appendTo(t *testing.T, path, extra string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, extra...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The preview renders on the server, through the same builder, theme and
// resolver a build uses. This is what that is worth: an author looking at the
// preview is looking at the page, not at an approximation of it that happens
// to live in the admin.
//
// A front end running its own markdown library would disagree about
// footnotes, highlighting, raw html and every extension either side adds
// later, and "the preview does not match the site" is a complaint with no end.
func TestPreviewOfSavedContentIsByteIdenticalToTheBuiltPage(t *testing.T) {
	root := newProject(t, 4)

	built := openSite(t, root)
	outDir := filepath.Join(root, "public")
	if _, _, err := built.Build(t.Context(), site.BuildOptions{OutDir: outDir, Now: frozen}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(outDir, "posts", "post-01", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	handler := newServer(t, root, serve.Options{Admin: true, Write: true}).Handler()

	// Read the item back the way the admin does, then preview exactly what
	// was read: unchanged content must render to the same bytes.
	item := readItem(t, handler, findID(t, handler, "post-01"))
	draft, err := json.Marshal(map[string]any{
		"kind":         item["kind"],
		"title":        item["title"],
		"slug":         item["slug"],
		"status":       item["status"],
		"body":         item["body"],
		"meta":         item["meta"],
		"taxonomies":   item["taxonomies"],
		"published_at": item["published_at"],
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/preview?id="+item["id"].(string), bytes.NewReader(draft))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("preview returned %d\n%s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), want) {
		t.Errorf("the preview differs from the built page\n%s", firstDifference(string(want), rec.Body.String()))
	}
}

func findID(t *testing.T, h http.Handler, slug string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/contents?limit=500", nil))

	var page struct {
		Items []struct{ ID, Slug string } `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.Slug == slug {
			return item.ID
		}
	}
	t.Fatalf("no item with slug %q", slug)
	return ""
}

func readItem(t *testing.T, h http.Handler, id string) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/contents/"+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: %d", id, rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// Changing a setting has to take effect without a restart, or the admin would
// report a title the site is not serving.
func TestAChangedSettingReachesTheServedPages(t *testing.T) {
	root := newProject(t, 2)
	srv := newServer(t, root, serve.Options{Admin: true, Write: true})
	handler := srv.Handler()

	before := renderHome(t, handler)
	if !strings.Contains(before, "Parity") {
		t.Fatalf("the home page does not carry the configured title:\n%s", before)
	}

	// Write the configuration the way anything outside the admin would.
	config, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(config), "title: Parity", "title: Renamed", 1)
	if updated == string(config) {
		t.Fatal("the fixture no longer carries the title this test edits")
	}
	if err := os.WriteFile(filepath.Join(root, "kite.yaml"), []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := srv.Reload(t.Context()); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	after := renderHome(t, handler)
	if strings.Contains(after, "Parity") {
		t.Errorf("the page still carries the old title:\n%s", after)
	}
	if !strings.Contains(after, "Renamed") {
		t.Errorf("the page does not carry the new title:\n%s", after)
	}

	// And the API agrees with what is being served.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/site", nil))
	if !strings.Contains(rec.Body.String(), "Renamed") {
		t.Errorf("the api still reports the old title: %s", rec.Body.String())
	}
}

func renderHome(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: %d", rec.Code)
	}
	return rec.Body.String()
}

// The studio on a public address with nobody guarding it is the one mistake
// that looks like success from the operator's side, so it is refused at the
// point the server is built rather than warned about in a log nobody reads.
func TestAnUnguardedStudioRefusesAnAddressOtherMachinesCanReach(t *testing.T) {
	root := newProject(t, 1)
	s := openSite(t, root)

	start := func(opts serve.Options) error {
		_, err := serve.NewWithClock(t.Context(), s, opts, func() time.Time { return frozen })
		return err
	}

	if err := start(serve.Options{Addr: "0.0.0.0:1717", Admin: true}); err == nil {
		t.Fatal("an open studio started on 0.0.0.0")
	}

	// The site itself is meant to be reachable; only the studio is not.
	if err := start(serve.Options{Addr: "0.0.0.0:1717"}); err != nil {
		t.Errorf("serving the site publicly was refused: %v", err)
	}
	if err := start(serve.Options{Addr: "127.0.0.1:1717", Admin: true}); err != nil {
		t.Errorf("serving the studio on localhost was refused: %v", err)
	}

	account, err := auth.SetPassword(root, "admin", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if err := start(serve.Options{Addr: "0.0.0.0:1717", Admin: true, Auth: auth.New(account)}); err != nil {
		t.Errorf("a guarded studio was refused a public address: %v", err)
	}
}

// The admin's own files stay readable without a session: they are how a
// person reaches the sign-in form in the first place.
func TestTheStudioItselfLoadsBeforeAnybodyHasSignedIn(t *testing.T) {
	root := newProject(t, 1)
	account, err := auth.SetPassword(root, "admin", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}

	h := newServer(t, root, serve.Options{Admin: true, Auth: auth.New(account)}).Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Errorf("the studio refused to load its own page: %d", rec.Code)
	}

	// What it then asks for does need a session.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/site", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/site without a session: %d, want 401", rec.Code)
	}
}

// Refusing to start is the right answer for a server nobody can configure. It
// is the wrong one for a container: there is no terminal in there to run
// `kite auth set-password` in, so a writable server comes up in setup and
// answers nothing else until somebody finishes it.
func TestAnUnguardedStudioThatCanWriteComesUpInSetupInstead(t *testing.T) {
	root := newProject(t, 1)
	srv := newServer(t, root, serve.Options{Addr: "0.0.0.0:1717", Admin: true, Write: true})

	if !srv.Setup().Pending() {
		t.Fatal("a server with no account is not waiting to be set up")
	}

	h := srv.Handler()
	status := func(path string) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}

	// The site is what the public came for, and it is not what setup guards.
	if got := status("/"); got != http.StatusOK {
		t.Errorf("GET /: %d, want 200", got)
	}
	// The studio's own page loads, because it is how the form is reached.
	if got := status("/admin/"); got == http.StatusUnauthorized || got == http.StatusForbidden {
		t.Errorf("GET /admin/: %d, want the installer to be reachable", got)
	}
	// What it holds does not.
	for _, path := range []string{"/api/v1/contents", "/api/v1/site"} {
		if got := status(path); got != http.StatusForbidden {
			t.Errorf("GET %s: %d, want 403", path, got)
		}
	}
	if got := status("/api/v1/setup"); got != http.StatusOK {
		t.Errorf("GET /api/v1/setup: %d, want 200", got)
	}
}

// A server with an account, or one on localhost, was never waiting for
// anything and must not be made to wait now.
func TestAServerThatNeedsNoSetupHasNoFlow(t *testing.T) {
	root := newProject(t, 1)

	for _, c := range []struct {
		name string
		opts serve.Options
	}{
		{"localhost", serve.Options{Addr: "127.0.0.1:1717", Admin: true, Write: true}},
		{"no studio", serve.Options{Addr: "0.0.0.0:1717"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if srv := newServer(t, root, c.opts); srv.Setup().Pending() {
				t.Error("this server is waiting to be set up and should not be")
			}
		})
	}

	account, err := auth.SetPassword(root, "admin", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	srv := newServer(t, root, serve.Options{
		Addr: "0.0.0.0:1717", Admin: true, Write: true, Auth: auth.New(account),
	})
	if srv.Setup().Pending() {
		t.Error("a server that already has an account is waiting to be set up")
	}
}
