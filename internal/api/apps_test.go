package api_test

import (
	"bytes"
	"crypto/rand"
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

	"aead.dev/minisign"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/apps"
	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/lock"
)

// pngBytes is enough of a PNG for a sniffer to know it.
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

// indexServer serves an index of themes and plugins and their archives, as
// kite-plus/apps and jsDelivr do.
type indexServer struct {
	t     *testing.T
	srv   *httptest.Server
	mu    sync.Mutex
	index apps.Index
	files map[string][]byte
	key   minisign.PrivateKey
	pub   minisign.PublicKey
}

func newIndexServer(t *testing.T) *indexServer {
	pub, key, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ix := &indexServer{t: t, index: apps.Index{Format: apps.Format}, key: key, pub: pub, files: map[string][]byte{
		"/paper.png":  pngBytes,
		"/paper.html": []byte("<!doctype html><script>alert(1)</script>"),
		"/greet.svg":  []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><circle cx="32" cy="32" r="16" fill="#4A77D6"/></svg>`),
		"/evil.svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`),
	}}
	ix.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ix.mu.Lock()
		defer ix.mu.Unlock()
		if r.URL.Path == "/index.json" || r.URL.Path == "/index.json.minisig" {
			data, err := json.Marshal(ix.index)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if r.URL.Path == "/index.json.minisig" {
				data = minisign.Sign(ix.key, data)
			}
			_, _ = w.Write(data)
			return
		}
		data, ok := ix.files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(ix.srv.Close)
	return ix
}

// publish lists a new version of a package, packed as its release carries it.
func (ix *indexServer) publish(kind, id, version string, files map[string]string) {
	ix.t.Helper()
	packed := make(map[string][]byte, len(files))
	for name, body := range files {
		packed[name] = []byte(body)
	}
	var buf bytes.Buffer
	if err := archive.Pack(&buf, id, packed); err != nil {
		ix.t.Fatal(err)
	}
	data := buf.Bytes()
	sum := sha256.Sum256(data)
	path := fmt.Sprintf("/%s-%s-%s.zip", kind, id, version)
	api := map[string]string{"theme": "kite/v1", "plugin": "kite/plugin/v1"}[kind]

	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.files[path] = data
	a := ix.index.Find(kind, id)
	if a == nil {
		a = &apps.App{Kind: kind, ID: id, Official: true, Repo: "kite-plus/" + id,
			Title: map[string]string{"en": strings.ToUpper(id[:1]) + id[1:], "zh-CN": "纸"}}
		ix.index.Apps = append(ix.index.Apps, a)
	}
	a.Versions = slices.Insert(a.Versions, 0, &apps.Release{
		Version: version, API: api, Notes: "https://example.com/" + version,
		Archive: apps.Archive{URLs: []string{ix.srv.URL + path}, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))},
	})
}

func (ix *indexServer) set(edit func(*apps.Index)) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	edit(&ix.index)
}

// withIndex gives a server's view the index at ix, kept in root's cache,
// and the site's kite.lock.
func withIndex(root string, ix *indexServer) func(*api.Options) {
	return func(o *api.Options) {
		inner := o.Site
		o.Site = func() api.View {
			v := inner()
			client, err := apps.ClientFor(root, ix.srv.URL+"/index.json", ix.pub.String())
			if err != nil {
				panic(err)
			}
			v.Apps = client
			v.Lock = func() (*lock.File, error) { return lock.Read(root) }
			v.PackageTree = func(kind, name string) (string, error) {
				return lock.TreeOf(filepath.Join(root, apps.Dir(kind, name)))
			}
			return v
		}
	}
}

func paperFiles(version string) map[string]string {
	return map[string]string{
		"theme.yaml":          "name: paper\ntitle: Paper\nversion: " + version + "\napiVersion: kite/v1\n",
		"layouts/single.html": "{{ .Site.Title }}",
	}
}

func greetFiles(version, extra string) map[string]string {
	return map[string]string{"plugin.yaml": "id: greet\nname: Greet\nversion: " + version +
		"\napiVersion: kite/plugin/v1\ninject:\n  - at: body\n    html: <p>hi</p>\n" + extra}
}

func appsOf(t *testing.T, h http.Handler, query string) map[string]api.AppInfo {
	t.Helper()
	list := getWith[api.AppList](t, h, api.Prefix+"/apps"+query, map[string]string{"Accept-Language": "zh-CN"})
	out := map[string]api.AppInfo{}
	for _, a := range list.Items {
		out[a.Kind+"/"+a.ID] = a
	}
	return out
}

func TestTheStudioInstallsAndUpdatesFromTheIndex(t *testing.T) {
	ix := newIndexServer(t)
	ix.publish("theme", "paper", "1.0.0", paperFiles("1.0.0"))
	ix.publish("plugin", "greet", "1.0.0", greetFiles("1.0.0", ""))
	ix.set(func(i *apps.Index) { i.Find("theme", "paper").Screenshot = ix.srv.URL + "/paper.png" })
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, withIndex(root, ix))

	listed := appsOf(t, h, "")
	paper := listed["theme/paper"]
	if paper.Title != "纸" || paper.Version != "1.0.0" || paper.Installed != "" || paper.Other ||
		paper.Screenshot != api.Prefix+"/apps/theme/paper/screenshot" || listed["plugin/greet"].Version != "1.0.0" {
		t.Fatalf("listed %+v", listed)
	}
	if only := appsOf(t, h, "?kind=plugin"); len(only) != 1 || only["plugin/greet"].ID == "" {
		t.Errorf("plugins only: %+v", only)
	}
	if found := appsOf(t, h, "?q=nothing+like+it"); len(found) != 0 {
		t.Errorf("a search for nothing found %+v", found)
	}
	rec := send(t, h, http.MethodGet, paper.Screenshot, nil, nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Errorf("screenshot: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}

	rec = send(t, h, http.MethodPost, api.Prefix+"/apps/theme/paper/install", nil, nil)
	if rec.Code != http.StatusCreated || decode[api.AppInfo](t, rec).Installed != "1.0.0" {
		t.Fatalf("install: %d %s", rec.Code, rec.Body.String())
	}
	if e, ok := readLockFile(t, root).Get("theme", "paper"); !ok || e.Version != "1.0.0" {
		t.Errorf("kite.lock records %+v", e)
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/apps/theme/paper/install", nil, nil); rec.Code != http.StatusConflict {
		t.Errorf("installing it again: %d", rec.Code)
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/apps/plugin/greet/install", api.AppInstall{Version: "1.0.0"}, nil); rec.Code != http.StatusCreated {
		t.Fatalf("install the plugin: %d %s", rec.Code, rec.Body.String())
	}

	ix.publish("theme", "paper", "1.1.0", paperFiles("1.1.0"))
	ix.publish("plugin", "greet", "1.1.0", greetFiles("1.1.0", "  - at: head\n    html: <script src=\"https://cdn.example.com/g.js\"></script>\n"))
	listed = appsOf(t, h, "?refresh=true")
	if listed["theme/paper"].Update != "1.1.0" || listed["plugin/greet"].Update != "1.1.0" {
		t.Fatalf("after the release: %+v", listed)
	}

	// A theme changed by hand is replaced only once that is agreed to.
	style := filepath.Join(root, "themes", "paper", "layouts", "single.html")
	if err := os.WriteFile(style, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := getWith[api.UpdatePlan](t, h, api.Prefix+"/apps/theme/paper/update", nil)
	if plan.From != "1.0.0" || plan.To != "1.1.0" || !plan.Changed || !strings.Contains(plan.Why, "changed since 1.0.0") || plan.Grows {
		t.Errorf("plan %+v", plan)
	}
	rec = send(t, h, http.MethodPost, api.Prefix+"/apps/theme/paper/update", api.AppUpdate{}, nil)
	if needs := decode[api.UpdateNeeds](t, rec); rec.Code != http.StatusConflict ||
		needs.Error.Code != api.CodeUpdateNeedsConfirmation || needs.Plan.To != "1.1.0" {
		t.Errorf("an update without agreement: %d %s", rec.Code, rec.Body.String())
	}
	rec = send(t, h, http.MethodPost, api.Prefix+"/apps/theme/paper/update", api.AppUpdate{Confirm: api.AppConfirm{Overwrite: true}}, nil)
	if rec.Code != http.StatusOK || decode[api.AppInfo](t, rec).Installed != "1.1.0" {
		t.Errorf("an agreed update: %d %s", rec.Code, rec.Body.String())
	}

	// A plugin that would load from another site is updated once agreed to.
	plan = getWith[api.UpdatePlan](t, h, api.Prefix+"/apps/plugin/greet/update", nil)
	if !plan.Grows || !slices.Equal(plan.MoreLoads, []string{"cdn.example.com"}) || len(plan.MoreHooks) != 0 ||
		plan.Changed || plan.Inject != 2 || plan.Injected != 1 {
		t.Errorf("plugin plan %+v", plan)
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/apps/plugin/greet/update", api.AppUpdate{}, nil); rec.Code != http.StatusConflict {
		t.Errorf("a plugin update without agreement: %d", rec.Code)
	}
	rec = send(t, h, http.MethodPost, api.Prefix+"/apps/plugin/greet/update", api.AppUpdate{Confirm: api.AppConfirm{Grant: true}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("an agreed plugin update: %d %s", rec.Code, rec.Body.String())
	}
	if e, _ := readLockFile(t, root).Get("plugin", "greet"); e.Version != "1.1.0" || e.Granted == nil || !slices.Equal(e.Granted.Loads, []string{"cdn.example.com"}) {
		t.Errorf("kite.lock records %+v", e)
	}
}

func TestAPictureThatIsNotOneIsRefused(t *testing.T) {
	ix := newIndexServer(t)
	ix.publish("theme", "paper", "1.0.0", paperFiles("1.0.0"))
	ix.set(func(i *apps.Index) { i.Find("theme", "paper").Screenshot = ix.srv.URL + "/paper.html" })
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, withIndex(root, ix))
	if rec := send(t, h, http.MethodGet, api.Prefix+"/apps/theme/paper/screenshot", nil, nil); rec.Code != http.StatusBadRequest ||
		strings.Contains(rec.Body.String(), "<script>") {
		t.Errorf("a page served as a picture: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTheIndexOutOfReachIsTheNetworksFault(t *testing.T) {
	ix := newIndexServer(t)
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, withIndex(root, ix))
	ix.srv.Close()
	rec := send(t, h, http.MethodGet, api.Prefix+"/apps", nil, nil)
	if rec.Code != http.StatusBadGateway || decode[api.ErrorBody](t, rec).Error.Code != api.CodeIndexUnreachable {
		t.Errorf("an index out of reach: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAReadOnlyServerOnlyBrowsesTheIndex(t *testing.T) {
	ix := newIndexServer(t)
	ix.publish("theme", "paper", "1.0.0", paperFiles("1.0.0"))
	root := newProject(t, 1)
	h, _ := newServer(t, root, withIndex(root, ix))
	if listed := appsOf(t, h, ""); listed["theme/paper"].Version != "1.0.0" {
		t.Errorf("listed %+v", listed)
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/apps/theme/paper/install", nil, nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("an install on a read-only server: %d", rec.Code)
	}
}

func readLockFile(t *testing.T, root string) *lock.File {
	t.Helper()
	f, err := lock.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAWronglySignedIndexIsTheIndexsFault(t *testing.T) {
	ix := newIndexServer(t)
	ix.publish("theme", "paper", "1.0.0", paperFiles("1.0.0"))
	_, stranger, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ix.mu.Lock()
	ix.key = stranger
	ix.mu.Unlock()
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, withIndex(root, ix))
	rec := send(t, h, http.MethodGet, api.Prefix+"/apps", nil, nil)
	if rec.Code != http.StatusBadGateway || decode[api.ErrorBody](t, rec).Error.Code != api.CodeIndexUntrusted {
		t.Errorf("a wrongly signed index: %d %s", rec.Code, rec.Body.String())
	}
}

// An icon may be an SVG, which the studio would run as itself if it were
// served plainly: it is served sandboxed, and one that would run something
// is not served at all.
func TestAnIconIsServedSoThatNothingInItRuns(t *testing.T) {
	ix := newIndexServer(t)
	ix.publish("plugin", "greet", "1.0.0", greetFiles("1.0.0", ""))
	ix.set(func(i *apps.Index) { i.Find("plugin", "greet").Icon = ix.srv.URL + "/greet.svg" })
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, withIndex(root, ix))

	greet := appsOf(t, h, "?kind=plugin")["plugin/greet"]
	if greet.Icon != api.Prefix+"/apps/plugin/greet/icon" {
		t.Fatalf("icon %q", greet.Icon)
	}
	rec := send(t, h, http.MethodGet, greet.Icon, nil, nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/svg+xml" ||
		!strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("icon: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Content-Security-Policy"))
	}

	ix.set(func(i *apps.Index) { i.Find("plugin", "greet").Icon = ix.srv.URL + "/evil.svg" })
	other := newProject(t, 1)
	evil, _ := newWritableServer(t, other, withIndex(other, ix))
	if rec := send(t, evil, http.MethodGet, greet.Icon, nil, nil); rec.Code == http.StatusOK ||
		strings.Contains(rec.Body.String(), "onload") {
		t.Errorf("an SVG with a script was served: %d %s", rec.Code, rec.Body.String())
	}
}
