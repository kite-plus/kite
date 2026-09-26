package api_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	kiteconfig "github.com/kite-plus/kite/internal/config"
)

const helloPlugin = `id: hello
name: Hello
version: 1.0.0
apiVersion: kite/plugin/v1
description: Says hello under every post.
inject:
  - at: body
    pages: [single]
    html: <script src="https://cdn.example.net/hello.js" data-greeting="{{ .Settings.greeting }}"></script>
settings:
  - {key: greeting, type: string, label: Greeting, default: hi}
  - {key: loud, type: boolean, label: Loud}
`

func installPlugin(t *testing.T, h http.Handler, archive []byte, query string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "plugin.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, api.Prefix+"/plugins"+query, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// settingsTag is the revision a settings change has to be made against.
func settingsTag(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /settings: %d", rec.Code)
	}
	return rec.Header().Get("ETag")
}

// A plugin arrives as an archive, is installed off, and is turned on and set
// up through the settings, which is what writes it into kite.yaml.
func TestAPluginIsInstalledOffAndTurnedOnThroughTheSettings(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)

	rec := installPlugin(t, h, zipOf(t,
		entry{name: "hello-1.0.0/plugin.yaml", body: helloPlugin},
		entry{name: "hello-1.0.0/assets/hello.css", body: "p {}"},
		entry{name: "__MACOSX/hello-1.0.0/._plugin.yaml", body: "junk"},
	), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("install: status %d\n%s", rec.Code, rec.Body.String())
	}
	info := decode[api.PluginInfo](t, rec)
	if info.ID != "hello" || info.Enabled || !info.Injects || !slices.Equal(info.Hosts, []string{"cdn.example.net"}) {
		t.Errorf("installed = %+v, want hello, off, injecting, loading from cdn.example.net", info)
	}
	for _, name := range []string{"plugin.yaml", "assets/hello.css"} {
		if _, err := os.Stat(filepath.Join(root, "plugins", "hello", filepath.FromSlash(name))); err != nil {
			t.Errorf("%s was not written: %v", name, err)
		}
	}

	rec = send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{
		"plugins.enabled":                 []string{"hello"},
		"plugins.settings.hello.greeting": "hey",
	}, map[string]string{"If-Match": settingsTag(t, h)})
	if rec.Code != http.StatusOK {
		t.Fatalf("turning it on: status %d\n%s", rec.Code, rec.Body.String())
	}
	cfg, err := kiteconfig.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.Plugins.Enabled, []string{"hello"}) || cfg.Plugins.Settings["hello"]["greeting"] != "hey" {
		t.Errorf("kite.yaml plugins = %+v", cfg.Plugins)
	}

	detail := getWith[api.PluginDetail](t, h, api.Prefix+"/plugins/hello", nil)
	// A switch with no default is absent, which the form reads as off.
	if !detail.Enabled || detail.Values["greeting"] != "hey" || detail.Values["loud"] == true {
		t.Errorf("detail = %+v, want it on with the stored greeting", detail)
	}
	if !slices.Equal(detail.Stored, []string{"greeting"}) || len(detail.Schema) != 2 {
		t.Errorf("stored = %q, schema of %d fields", detail.Stored, len(detail.Schema))
	}

	list := getWith[api.List[api.PluginInfo]](t, h, api.Prefix+"/plugins", nil)
	if len(list.Items) != 1 || !list.Items[0].Enabled {
		t.Errorf("list = %+v", list.Items)
	}
}

func TestAPluginArchiveIsCheckedAsASiteWouldCheckIt(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)

	for name, archive := range map[string][]byte{
		"no manifest":    zipOf(t, entry{name: "readme.md", body: "hello"}),
		"a bad manifest": zipOf(t, entry{name: "plugin.yaml", body: "id: hello\nname: Hello\nversion: 1\napiVersion: kite/plugin/v1\n"}),
		"no module":      zipOf(t, entry{name: "plugin.yaml", body: helloPlugin + "hooks: [transform_html]\n"}),
		"not a zip":      []byte("plain text"),
	} {
		if rec := installPlugin(t, h, archive, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400\n%s", name, rec.Code, rec.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "plugins")); !os.IsNotExist(err) {
		t.Error("a refused archive left something in plugins/")
	}

	good := zipOf(t, entry{name: "plugin.yaml", body: helloPlugin})
	if rec := installPlugin(t, h, good, ""); rec.Code != http.StatusCreated {
		t.Fatalf("install: %d", rec.Code)
	}
	newer := zipOf(t, entry{name: "plugin.yaml", body: strings.Replace(helloPlugin, "version: 1.0.0", "version: 1.1.0", 1)})
	rec := installPlugin(t, h, newer, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("a second install: status %d, want 409", rec.Code)
	}
	exists := decode[api.PluginExists](t, rec)
	if exists.Installed.Version != "1.0.0" || exists.Uploaded.Version != "1.1.0" {
		t.Errorf("conflict = %+v, want both versions", exists)
	}
	if rec := installPlugin(t, h, newer, "?replace=true"); rec.Code != http.StatusCreated {
		t.Errorf("replace: status %d", rec.Code)
	}
}

// Turning on a plugin that cannot load would stop the next build, and a
// setting it does not declare would sit in kite.yaml doing nothing, so both
// are refused before anything is written.
func TestSettingsRefuseWhatAPluginCannotTake(t *testing.T) {
	root := newProject(t, 1)
	write(t, filepath.Join(root, "plugins", "hello", "plugin.yaml"), helloPlugin)
	write(t, filepath.Join(root, "plugins", "broken", "plugin.yaml"), "id: broken\n")
	h, _ := newWritableServer(t, root)

	for _, tc := range []struct {
		name   string
		values map[string]any
		field  string
	}{
		{"not installed", map[string]any{"plugins.enabled": []string{"ghost"}}, "plugins.enabled"},
		{"cannot load", map[string]any{"plugins.enabled": []string{"broken"}}, "plugins.enabled"},
		{"twice", map[string]any{"plugins.enabled": []string{"hello", "hello"}}, "plugins.enabled"},
		{"an unknown setting", map[string]any{"plugins.settings.hello.volume": 11}, "plugins.settings.hello.volume"},
		{"the wrong kind", map[string]any{"plugins.settings.hello.loud": "yes"}, "plugins.settings.hello.loud"},
		{"an unknown plugin", map[string]any{"plugins.settings.ghost.x": 1}, "plugins.settings.ghost.x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPut, api.Prefix+"/settings", tc.values,
				map[string]string{"If-Match": settingsTag(t, h)})
			if rec.Code != http.StatusBadRequest || failureOf(t, rec).Field != tc.field {
				t.Errorf("status %d, body %s; want 400 on %s", rec.Code, rec.Body.String(), tc.field)
			}
		})
	}
	if cfg, err := kiteconfig.Load(root); err != nil || len(cfg.Plugins.Enabled) != 0 || len(cfg.Plugins.Settings) != 0 {
		t.Errorf("a refused change was written: %+v, %v", cfg.Plugins, err)
	}
}

func TestAPluginThatIsOnIsNotRemoved(t *testing.T) {
	root := newProject(t, 1)
	write(t, filepath.Join(root, "plugins", "hello", "plugin.yaml"), helloPlugin)
	h, _ := newWritableServer(t, root)

	if rec := send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{"plugins.enabled": []string{"hello"}},
		map[string]string{"If-Match": settingsTag(t, h)}); rec.Code != http.StatusOK {
		t.Fatalf("turning it on: %d", rec.Code)
	}
	if rec := send(t, h, http.MethodDelete, api.Prefix+"/plugins/hello", nil, nil); rec.Code != http.StatusConflict {
		t.Errorf("removing a plugin that is on: status %d, want 409", rec.Code)
	}

	if rec := send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{"plugins.enabled": []string{}},
		map[string]string{"If-Match": settingsTag(t, h)}); rec.Code != http.StatusOK {
		t.Fatalf("turning it off: %d", rec.Code)
	}
	if rec := send(t, h, http.MethodDelete, api.Prefix+"/plugins/hello", nil, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("removing it: status %d\n%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "plugins", "hello")); !os.IsNotExist(err) {
		t.Error("the plugin's directory is still there")
	}
}
