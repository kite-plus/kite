package serve_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/plugin/plugintest"
	"github.com/kite-plus/kite/internal/serve"
	"github.com/kite-plus/kite/internal/site"
)

const hello = `id: hello
name: Hello
version: 1.0.0
apiVersion: kite/plugin/v1
inject:
  - at: head
    html: <link rel="stylesheet" href="{{ asset "hello.css" }}">
  - at: body
    pages: [single]
    kinds: [post]
    html: <p class="hello" data-greeting="{{ .Settings.greeting }}" data-page="{{ .Page.ID }}"></p>
settings:
  - key: greeting
    type: string
    default: hi
`

// installHello puts the hello plugin into a project and turns it on, with
// settings written as the lines under plugins.settings.
func installHello(t *testing.T, root, settings string) {
	t.Helper()
	dir := filepath.Join(root, "plugins", "hello")
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"plugin.yaml": hello, "assets/hello.css": ".hello { color: teal }\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	appendConfig(t, root, "plugins:\n  enabled: [hello]\n"+settings)
}

func appendConfig(t *testing.T, root, text string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(root, "kite.yaml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

func fetch(t *testing.T, h http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec
}

// A plugin's code goes into the pages it covers and its files are published
// beside them, in a build and in a preview alike: a plugin that looked right
// in the studio and differed once published would be worse than none.
func TestAPluginReachesBuiltAndServedPagesAlike(t *testing.T) {
	root := newProjectAt(t, 3, "https://example.github.io/blog/")
	installHello(t, root, "  settings:\n    hello: {greeting: 你好}\n")

	out := filepath.Join(root, "public")
	if _, _, err := openSite(t, root).Build(t.Context(), site.BuildOptions{OutDir: out, Now: frozen}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	if css := read("plugins/hello/hello.css"); !strings.Contains(css, "teal") {
		t.Errorf("the plugin's stylesheet was not published: %q", css)
	}
	post := read("posts/post-00/index.html")
	if !strings.Contains(post, `<link rel="stylesheet" href="/blog/plugins/hello/hello.css">`) {
		t.Error("the post does not link the plugin's stylesheet under the site's path")
	}
	if !strings.Contains(post, `<p class="hello" data-greeting="你好" data-page="01J8KQ2P3R4S5T6V7W8X9YZ000"></p>`) {
		t.Error("the post does not carry the plugin's code with the site's settings")
	}
	if about := read("about/index.html"); strings.Contains(about, `class="hello"`) {
		t.Error("a page that is not a post got the code meant for posts")
	}
	if home := read("index.html"); !strings.Contains(home, "hello.css") || strings.Contains(home, `class="hello"`) {
		t.Error("the home page should link the stylesheet and nothing more")
	}

	handler := newServer(t, root, serve.Options{}).Handler()
	for _, rel := range []string{"posts/post-00/index.html", "index.html", "about/index.html", "plugins/hello/hello.css"} {
		rec := fetch(t, handler, "/blog/"+rel)
		if rec.Code != http.StatusOK {
			t.Errorf("/blog/%s: status %d", rel, rec.Code)
			continue
		}
		if want := read(rel); rec.Body.String() != want {
			t.Errorf("/blog/%s differs between build and serve\n%s", rel, firstDifference(want, rec.Body.String()))
		}
	}
}

// A plugin's module rewrites sources and pages and writes files of its own,
// the same in a build as in a preview.
func TestAModuleReachesBuiltAndServedPagesAlike(t *testing.T) {
	root := newProjectAt(t, 3, "https://example.github.io/blog/")
	dir := filepath.Join(root, "plugins", "guest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := plugintest.Manifest("guest", plugin.HookTransformMarkdown, plugin.HookTransformHTML, plugin.HookBuildComplete)
	for name, body := range map[string][]byte{"plugin.yaml": []byte(manifest), "plugin.wasm": plugintest.Wasm(t)} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	about := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ900\ntitle: About\nslug: about\nstatus: published\n---\n\nFlying :kite:.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "pages", "about.md"), []byte(about), 0o644); err != nil {
		t.Fatal(err)
	}
	appendConfig(t, root, "plugins:\n  enabled: [guest]\n  settings:\n    guest: {sign: built}\n")

	out := filepath.Join(root, "public")
	if _, _, err := openSite(t, root).Build(t.Context(), site.BuildOptions{OutDir: out, Now: frozen}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	if page := read("about/index.html"); !strings.Contains(page, "Flying a kite at /blog/about/.") {
		t.Error("the page's source was not rewritten by the module")
	}
	if post := read("posts/post-00/index.html"); !strings.Contains(post, `<meta name="signed" content="built single /blog/posts/post-00/">`) {
		t.Errorf("the post was not rewritten by the module with the site's settings:\n%s", post)
	}
	var index []struct{ URL, Title, Text string }
	if err := json.Unmarshal([]byte(read("plugins/guest/index.json")), &index); err != nil {
		t.Fatal(err)
	}
	if len(index) != 4 {
		t.Errorf("index = %+v, want the three posts and the page", index)
	}

	handler := newServer(t, root, serve.Options{}).Handler()
	for _, rel := range []string{"about/index.html", "posts/post-00/index.html", "index.html", "plugins/guest/index.json"} {
		rec := fetch(t, handler, "/blog/"+rel)
		if rec.Code != http.StatusOK {
			t.Errorf("/blog/%s: status %d", rel, rec.Code)
			continue
		}
		if want := read(rel); rec.Body.String() != want {
			t.Errorf("/blog/%s differs between build and serve\n%s", rel, firstDifference(want, rec.Body.String()))
		}
	}
}

// A plugin that does not load leaves the preview working, so that the studio
// is there to fix it in, and stops a build, so that no site goes out without
// a plugin its author was counting on.
func TestAPluginThatDoesNotLoadStopsABuildButNotAPreview(t *testing.T) {
	root := newProject(t, 2)
	appendConfig(t, root, "plugins: [ghost]\n")

	s := openSite(t, root)
	if len(s.PluginProblems) != 1 || !strings.Contains(s.PluginProblems[0], "ghost") {
		t.Fatalf("problems = %q, want one naming ghost", s.PluginProblems)
	}
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: filepath.Join(root, "public"), Now: frozen}); err == nil ||
		!strings.Contains(err.Error(), "ghost") {
		t.Errorf("Build = %v, want a refusal naming the plugin", err)
	}
	if rec := fetch(t, newServer(t, root, serve.Options{}).Handler(), "/"); rec.Code != http.StatusOK {
		t.Errorf("the preview answered %d", rec.Code)
	}
}

func TestAnEditedPluginIsPickedUpWithoutARestart(t *testing.T) {
	root := newProject(t, 2)
	installHello(t, root, "")
	srv := newServer(t, root, serve.Options{})

	if body := fetch(t, srv.Handler(), "/posts/post-00/").Body.String(); !strings.Contains(body, `data-greeting="hi"`) {
		t.Fatalf("the default greeting is not on the page:\n%s", body)
	}
	manifest := filepath.Join(root, "plugins", "hello", "plugin.yaml")
	if err := os.WriteFile(manifest, []byte(strings.Replace(hello, "default: hi", "default: hey there", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := srv.Reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	if body := fetch(t, srv.Handler(), "/posts/post-00/").Body.String(); !strings.Contains(body, `data-greeting="hey there"`) {
		t.Errorf("the edited plugin was not picked up:\n%s", body)
	}
}
