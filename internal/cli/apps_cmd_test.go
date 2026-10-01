package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/kite-plus/kite/internal/apps"
	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/lock"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/render/theme"
)

// fakeIndex serves an index of themes and plugins, and their archives, as
// kite-plus/apps and jsDelivr do, and is the index KITE_APPS_URL names.
type fakeIndex struct {
	t     *testing.T
	srv   *httptest.Server
	mu    sync.Mutex
	index apps.Index
	files map[string][]byte
}

func newFakeIndex(t *testing.T) *fakeIndex {
	f := &fakeIndex{t: t, index: apps.Index{Format: apps.Format}, files: map[string][]byte{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path == "/index.json" {
			_ = json.NewEncoder(w).Encode(f.index)
			return
		}
		data, ok := f.files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(f.srv.Close)
	t.Setenv("KITE_APPS_URL", f.srv.URL+"/index.json")
	return f
}

// publish lists a new version of a package, packed as its release carries
// it.
func (f *fakeIndex) publish(kind, id, version string, files map[string]string) {
	f.t.Helper()
	packed := make(map[string][]byte, len(files))
	for name, body := range files {
		packed[name] = []byte(body)
	}
	var buf bytes.Buffer
	if err := archive.Pack(&buf, id, packed); err != nil {
		f.t.Fatal(err)
	}
	data := buf.Bytes()
	sum := sha256.Sum256(data)
	path := fmt.Sprintf("/%s-%s-%s/%s-%s.zip", kind, id, version, id, version)
	api := theme.APIVersion
	if kind == "plugin" {
		api = plugin.APIVersion
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = data
	app := f.index.Find(kind, id)
	if app == nil {
		app = &apps.App{Kind: kind, ID: id, Repo: "someone/" + id, Title: map[string]string{"en": strings.ToUpper(id[:1]) + id[1:]}}
		f.index.Apps = append(f.index.Apps, app)
	}
	app.Versions = slices.Insert(app.Versions, 0, &apps.Release{
		Version: version,
		API:     api,
		Archive: apps.Archive{URLs: []string{f.srv.URL + path}, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))},
	})
}

func paperTheme(version string) map[string]string {
	files := starterTheme("paper")
	files[theme.ManifestName] = strings.Replace(files[theme.ManifestName], "version: 0.1.0", "version: "+version, 1)
	return files
}

func greetPlugin(version, extra string) map[string]string {
	manifest := strings.Replace(starterPlugin("greet"), "version: 0.1.0", "version: "+version, 1)
	manifest = strings.Replace(manifest, "inject:\n", "inject:\n"+extra, 1)
	return map[string]string{plugin.ManifestName: manifest, "assets/greet.css": ".greet { font-style: italic; }\n"}
}

func readLock(t *testing.T, root string) *lock.File {
	t.Helper()
	f, err := lock.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAThemeAndAPluginComeFromTheIndexByNameAndAreKeptUpToDate(t *testing.T) {
	ix := newFakeIndex(t)
	ix.publish("theme", "paper", "1.0.0", paperTheme("1.0.0"))
	ix.publish("plugin", "greet", "1.0.0", greetPlugin("1.0.0", ""))
	site := newSite(t)
	writePost(t, site, "01J8KQ2P3R4S5T6V7W8X9YZ000", "hello", "status: published\npublished_at: 2026-01-01T00:00:00Z\n")

	if out := runKite(t, site, "theme", "add", "paper"); !strings.Contains(out, "installed paper 1.0.0 from the index in "+filepath.Join("themes", "paper")) {
		t.Errorf("theme add said:\n%s", out)
	}
	if out := runKite(t, site, "plugin", "add", "greet@1.0.0"); !strings.Contains(out, "it injects 2 piece(s) of code; its code loads nothing from other sites") {
		t.Errorf("plugin add said:\n%s", out)
	}
	lf := readLock(t, site)
	paper, ok := lf.Get("theme", "paper")
	if !ok || paper.Version != "1.0.0" || paper.Source != ix.srv.URL+"/index.json" || !strings.HasPrefix(paper.Checksum, "sha256:") {
		t.Fatalf("kite.lock records %+v", lf)
	}
	if greet, _ := lf.Get("plugin", "greet"); greet.Granted == nil || greet.Granted.Inject != 2 || len(greet.Granted.Loads) != 0 {
		t.Errorf("the plugin's grant is %+v", greet.Granted)
	}
	runKite(t, site, "theme", "use", "paper")
	runKite(t, site, "plugin", "enable", "greet")
	runKite(t, site, "build")
	if out := runKite(t, site, "apps", "outdated"); !strings.Contains(out, "up to date") {
		t.Errorf("outdated before any update:\n%s", out)
	}
	if out := runKite(t, site, "apps", "search", "pap"); !strings.Contains(out, "paper") || !strings.Contains(out, "installed 1.0.0") ||
		strings.Contains(out, "greet") {
		t.Errorf("search said:\n%s", out)
	}

	ix.publish("theme", "paper", "1.1.0", paperTheme("1.1.0"))
	ix.publish("plugin", "greet", "1.1.0", greetPlugin("1.1.0",
		"  - at: head\n    html: <script src=\"https://cdn.example.com/greet.js\"></script>\n"))
	out := runKite(t, site, "apps", "outdated", "--refresh")
	for _, want := range []string{"paper            1.0.0    -> 1.1.0", "greet            1.0.0    -> 1.1.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("outdated does not say %q:\n%s", want, out)
		}
	}

	// A theme changed by hand is not replaced, and a plugin that would load
	// from another site is not updated, without being told to.
	style := filepath.Join(site, "themes", "paper", "static", "style.css")
	mine, err := os.ReadFile(style)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(style, append(mine, "/* mine */\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := runKite(t, site, "doctor"); !strings.Contains(out, filepath.Join("themes", "paper")+" has changed since paper 1.0.0") {
		t.Errorf("doctor said:\n%s", out)
	}
	out, err = tryKite(t, site, "apps", "update")
	if err == nil || !strings.Contains(out, "Run with --force") ||
		!strings.Contains(out, "plugin greet 1.1.0 loads from cdn.example.com, injects 3 pieces of code, not 2; run with --yes") {
		t.Errorf("update = %v\n%s", err, out)
	}
	if v, _ := readLock(t, site).Get("plugin", "greet"); v.Version != "1.0.0" {
		t.Errorf("the plugin was updated without agreement: %+v", v)
	}

	out = runKite(t, site, "apps", "update", "--force", "--yes")
	if !strings.Contains(out, "updated theme paper from 1.0.0 to 1.1.0") || !strings.Contains(out, "updated plugin greet from 1.0.0 to 1.1.0") {
		t.Errorf("update said:\n%s", out)
	}
	lf = readLock(t, site)
	if greet, _ := lf.Get("plugin", "greet"); greet.Version != "1.1.0" || greet.Granted == nil || !slices.Equal(greet.Granted.Loads, []string{"cdn.example.com"}) {
		t.Errorf("after the update kite.lock records %+v", greet)
	}
	if out := runKite(t, site, "doctor"); strings.Contains(out, "has changed") {
		t.Errorf("doctor after the update:\n%s", out)
	}
	runKite(t, site, "build")

	if _, err := tryKite(t, site, "theme", "add", "nowhere"); err == nil || !strings.Contains(err.Error(), "the index has no theme named nowhere") {
		t.Errorf("an unknown name: %v", err)
	}
	if _, err := tryKite(t, site, "apps", "update", "default"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Errorf("updating what is not installed: %v", err)
	}

	// With the index out of reach, the copy kept is used and says so.
	ix.srv.Close()
	if out := runKite(t, site, "apps", "search", "--refresh"); !strings.Contains(out, "using the copy fetched") || !strings.Contains(out, "paper") {
		t.Errorf("offline search said:\n%s", out)
	}
}

// A theme installed from its release's archive before the index was used is
// the same project when its homepage is the repository the index names, and
// an update first checks that its files are that release's.
func TestAThemeFromAnArchiveIsUpdatedWhenItIsTheIndexOwn(t *testing.T) {
	ix := newFakeIndex(t)
	files := paperTheme("1.0.0")
	files[theme.ManifestName] += "homepage: https://github.com/someone/paper\n"
	ix.publish("theme", "paper", "1.0.0", files)
	site := newSite(t)
	work := t.TempDir()
	writeFiles(t, filepath.Join(work, "paper"), files)
	runKite(t, site, "theme", "add", filepath.Join(work, "paper"))
	if lf := readLock(t, site); !lf.Empty() {
		t.Fatalf("an install from a folder was recorded: %+v", lf)
	}

	next := paperTheme("1.1.0")
	next[theme.ManifestName] += "homepage: https://github.com/someone/paper\n"
	ix.publish("theme", "paper", "1.1.0", next)
	if out := runKite(t, site, "apps", "outdated", "--refresh"); !strings.Contains(out, "-> 1.1.0  (installed from an archive)") {
		t.Errorf("outdated said:\n%s", out)
	}
	if out := runKite(t, site, "apps", "update", "paper"); !strings.Contains(out, "updated theme paper from 1.0.0 to 1.1.0") {
		t.Errorf("update said:\n%s", out)
	}
	if e, ok := readLock(t, site).Get("theme", "paper"); !ok || e.Version != "1.1.0" {
		t.Errorf("kite.lock records %+v", e)
	}
}

// A version released since the index was last fetched is found by asking for
// the index again, rather than after the hour the copy is kept.
func TestAVersionNewerThanTheKeptIndexIsFound(t *testing.T) {
	ix := newFakeIndex(t)
	ix.publish("theme", "paper", "1.0.0", paperTheme("1.0.0"))
	site := newSite(t)
	runKite(t, site, "apps", "search")
	ix.publish("theme", "paper", "1.1.0", paperTheme("1.1.0"))
	if out := runKite(t, site, "theme", "add", "paper@1.1.0"); !strings.Contains(out, "installed paper 1.1.0 from the index") {
		t.Errorf("theme add said:\n%s", out)
	}
	if _, err := tryKite(t, site, "theme", "add", "paper@9.0.0", "--replace"); err == nil ||
		!strings.Contains(err.Error(), "theme paper has no version 9.0.0; the index lists 1.1.0, 1.0.0") {
		t.Errorf("a version nobody released: %v", err)
	}
}
