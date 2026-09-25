package api_test

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

const paperManifest = `name: paper
title: Paper
version: 1.2.0
apiVersion: kite/v1
description: A quiet theme.
author: {name: Someone, url: https://example.com}
settings:
  - key: look
    type: section
    label: Look
    fields:
      - {key: accent, type: color, label: Accent, default: "#2563eb"}
  - {key: columns, type: number, label: Columns, default: 1, min: 1, max: 3}
`

// withThemes adds a working theme with a Chinese pack and a screenshot, and a
// theme written for a contract this Kite does not implement.
func withThemes(t *testing.T, root string) {
	t.Helper()
	for name, body := range map[string]string{
		"themes/paper/theme.yaml":          paperManifest,
		"themes/paper/layouts/single.html": "{{ .Site.ThemeSettings.accent }}",
		"themes/paper/i18n/zh-CN.yaml":     "theme:\n  title: 纸\n  settings:\n    accent: {label: 强调色}\n",
		"themes/paper/screenshot.png":      "pretend png",
		"themes/future/theme.yaml":         "name: future\nversion: 1.0.0\napiVersion: kite/v9\n",
	} {
		write(t, filepath.Join(root, filepath.FromSlash(name)), body)
	}
}

func getWith[T any](t *testing.T, h http.Handler, path string, headers map[string]string) T {
	t.Helper()
	rec := send(t, h, http.MethodGet, path, nil, headers)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d\n%s", path, rec.Code, rec.Body.String())
	}
	return decode[T](t, rec)
}

// Every theme a project has is listed, the one in use marked, and the ones
// that cannot be used with the reason; what a theme says about itself comes
// in the language the admin asks for, where the theme speaks it.
func TestThemesAreListedInTheLanguageAsked(t *testing.T) {
	root := newProject(t, 1)
	withThemes(t, root)
	h, _ := newWritableServer(t, root)

	themes := getWith[api.List[api.ThemeInfo]](t, h, api.Prefix+"/themes",
		map[string]string{"Accept-Language": "fr;q=0.5, zh-CN, en;q=0.8"}).Items
	byName := map[string]api.ThemeInfo{}
	for _, one := range themes {
		byName[one.Name] = one
	}

	if first := themes[0]; first.Name != "default" || !first.Builtin || !first.Active {
		t.Errorf("first = %+v, want the built-in theme, in use", first)
	}
	paper := byName["paper"]
	if paper.Title != "纸" || paper.Version != "1.2.0" || paper.Active || paper.Problem != "" {
		t.Errorf("paper = %+v", paper)
	}
	if paper.Author == nil || paper.Author.Name != "Someone" {
		t.Errorf("author = %+v", paper.Author)
	}
	if paper.Screenshot == "" {
		t.Fatal("paper has no screenshot address")
	}
	rec := send(t, h, http.MethodGet, paper.Screenshot, nil, nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "pretend png" || rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("screenshot: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if future := byName["future"]; !strings.Contains(future.Problem, "kite/v9") {
		t.Errorf("future = %+v, want it said to be written for another contract", future)
	}

	english := get[api.List[api.ThemeInfo]](t, h, api.Prefix+"/themes", http.StatusOK).Items
	for _, one := range english {
		if one.Name == "paper" && one.Title != "Paper" {
			t.Errorf("without a language asked for, title = %q", one.Title)
		}
	}
}

// A theme's own description carries what it can be configured with and what
// it would read from kite.yaml, so it can be configured before it is used.
func TestAThemeIsDescribedWithTheSettingsItWouldRead(t *testing.T) {
	root := newProject(t, 1)
	withThemes(t, root)
	write(t, filepath.Join(root, "kite.yaml"), config+"theme:\n  settings:\n    accent: \"#a3473b\"\n    nav: \"About | /about/\"\n")
	h, _ := newWritableServer(t, root)

	rec := send(t, h, http.MethodGet, api.Prefix+"/themes/paper", nil, map[string]string{"Accept-Language": "zh-CN"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") == "" {
		t.Error("no ETag, so settings saved from this description have no revision to go back with")
	}
	paper := decode[api.ThemeDetail](t, rec)
	if paper.Values["accent"] != "#a3473b" || paper.Values["columns"] != float64(1) {
		t.Errorf("values = %v, want the stored accent and the default columns", paper.Values)
	}
	if !slices.Equal(paper.Stored, []string{"accent"}) {
		t.Errorf("stored = %v, want only what kite.yaml holds and paper declares", paper.Stored)
	}
	if len(paper.Schema) != 2 || paper.Schema[0].Fields[0].Label != "强调色" {
		t.Errorf("schema = %+v", paper.Schema)
	}

	send404 := send(t, h, http.MethodGet, api.Prefix+"/themes/missing", nil, nil)
	if send404.Code != http.StatusNotFound {
		t.Errorf("a missing theme: status = %d", send404.Code)
	}
}

// Switching is checked before kite.yaml is touched, and so are the settings
// sent with it, against the theme they will belong to.
func TestSwitchingThemesIsCheckedBeforeAnythingIsWritten(t *testing.T) {
	root := newProject(t, 1)
	withThemes(t, root)
	h, _ := newWritableServer(t, root)
	configPath := filepath.Join(root, "kite.yaml")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	for _, tc := range []struct {
		name   string
		values map[string]any
		field  string
	}{
		{"a theme that is not installed", map[string]any{"theme.name": "missing"}, "theme.name"},
		{"a theme that cannot be used", map[string]any{"theme.name": "future"}, "theme.name"},
		{"a name that is a path", map[string]any{"theme.name": "../paper"}, "theme.name"},
		{"a setting the new theme lacks", map[string]any{"theme.name": "paper", "theme.settings.show_toc": true}, "theme.settings.show_toc"},
		{"a setting out of range", map[string]any{"theme.name": "paper", "theme.settings.columns": 9}, "theme.settings.columns"},
		{"a color that is not one", map[string]any{"theme.settings.accent": "brown"}, "theme.settings.accent"},
	} {
		rec := send(t, h, http.MethodPut, api.Prefix+"/settings", tc.values, map[string]string{"If-Match": tag})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400\n%s", tc.name, rec.Code, rec.Body.String())
			continue
		}
		if field := decode[api.ErrorBody](t, rec).Error.Field; field != tc.field {
			t.Errorf("%s: field = %q, want %q", tc.name, field, tc.field)
		}
	}
	if after, _ := os.ReadFile(configPath); string(after) != string(before) {
		t.Fatalf("a refused switch was written:\n%s", after)
	}

	rec := send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{
		"theme.name":             "paper",
		"theme.settings.columns": 2,
	}, map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("switch: status = %d\n%s", rec.Code, rec.Body.String())
	}
	settings := decode[api.Settings](t, rec)
	if settings.Theme.Name != "paper" || settings.Theme.Values["columns"] != float64(2) {
		t.Errorf("after the switch: %+v", settings.Theme)
	}
	themes := get[api.List[api.ThemeInfo]](t, h, api.Prefix+"/themes", http.StatusOK).Items
	for _, one := range themes {
		if one.Active != (one.Name == "paper") {
			t.Errorf("%s active = %v", one.Name, one.Active)
		}
	}
}

// Putting a setting back to the theme's default takes it out of kite.yaml.
func TestANullSettingGoesBackToTheDefault(t *testing.T) {
	root := newProject(t, 1)
	write(t, filepath.Join(root, "kite.yaml"), config+"theme:\n  settings:\n    show_toc: false\n    retired: x\n")
	h, _ := newWritableServer(t, root)
	tag := send(t, h, http.MethodGet, api.Prefix+"/settings", nil, nil).Header().Get("ETag")

	rec := send(t, h, http.MethodPut, api.Prefix+"/settings", map[string]any{
		"theme.settings.show_toc": nil,
		// A key the theme no longer declares can still be cleared away.
		"theme.settings.retired": nil,
	}, map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}
	if got := decode[api.Settings](t, rec).Theme.Values["show_toc"]; got != true {
		t.Errorf("show_toc = %v, want the theme's default", got)
	}
	data, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "show_toc") || strings.Contains(string(data), "retired") {
		t.Errorf("the keys are still in the file:\n%s", data)
	}
}

type entry struct {
	name, body string
	mode       fs.FileMode
}

func zipOf(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		header := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			header.SetMode(e.mode)
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func install(t *testing.T, h http.Handler, archive []byte, query string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "theme.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, api.Prefix+"/themes"+query, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// An archive of a repository holds the theme in one folder, with whatever
// the computer that made it added on its own; the theme is found in it and
// installed under its own name.
func TestAThemeIsInstalledFromAnArchive(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)

	rec := install(t, h, zipOf(t,
		entry{name: "paper-main/"},
		entry{name: "paper-main/theme.yaml", body: paperManifest},
		entry{name: "paper-main/layouts/single.html", body: "v1"},
		entry{name: "paper-main/layouts/old.html", body: "only in v1"},
		entry{name: "__MACOSX/paper-main/._theme.yaml", body: "junk"},
		entry{name: "paper-main/.DS_Store", body: "junk"},
	), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}
	if info := decode[api.ThemeInfo](t, rec); info.Name != "paper" || info.Version != "1.2.0" || info.Problem != "" {
		t.Errorf("installed = %+v", info)
	}
	for _, name := range []string{"theme.yaml", "layouts/single.html", "layouts/old.html"} {
		if _, err := os.Stat(filepath.Join(root, "themes", "paper", filepath.FromSlash(name))); err != nil {
			t.Errorf("%s was not installed: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "themes", "paper", ".DS_Store")); err == nil {
		t.Error("the Finder's file was installed with the theme")
	}

	// The same theme again is a question, answered with both versions.
	v2 := strings.Replace(paperManifest, "1.2.0", "1.3.0", 1)
	again := zipOf(t,
		entry{name: "theme.yaml", body: v2},
		entry{name: "layouts/single.html", body: "v2"},
	)
	rec = install(t, h, again, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("second install: status = %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	exists := decode[api.ThemeExists](t, rec)
	if exists.Installed.Version != "1.2.0" || exists.Uploaded.Version != "1.3.0" || exists.Error.Code != api.CodeConflict {
		t.Errorf("409 body = %+v", exists)
	}

	rec = install(t, h, again, "?replace=true")
	if rec.Code != http.StatusCreated {
		t.Fatalf("replace: status = %d\n%s", rec.Code, rec.Body.String())
	}
	if data, _ := os.ReadFile(filepath.Join(root, "themes", "paper", "layouts", "single.html")); string(data) != "v2" {
		t.Errorf("single.html = %q, want the new version", data)
	}
	if _, err := os.Stat(filepath.Join(root, "themes", "paper", "layouts", "old.html")); err == nil {
		t.Error("a template the new version dropped is still installed")
	}
}

// What goes into themes/ is checked the way a site checks a theme when it
// loads one, before anything is written, and an archive cannot put a file
// anywhere but inside the theme's own directory.
func TestAnArchiveThatIsNotAUsableThemeIsRefused(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)
	layout := entry{name: "layouts/single.html", body: "x"}

	for _, tc := range []struct {
		name    string
		archive []byte
	}{
		{"not an archive", []byte("plain text")},
		{"no theme.yaml", zipOf(t, layout)},
		{"two folders", zipOf(t, entry{name: "a/theme.yaml", body: paperManifest}, entry{name: "b/x.html", body: "x"})},
		{"a path out of the theme", zipOf(t, entry{name: "theme.yaml", body: paperManifest}, layout, entry{name: "../../escaped.html", body: "x"})},
		{"a link", zipOf(t, entry{name: "theme.yaml", body: paperManifest}, layout, entry{name: "layouts/list.html", body: "/etc/passwd", mode: fs.ModeSymlink | 0o777})},
		{"another contract", zipOf(t, entry{name: "theme.yaml", body: strings.Replace(paperManifest, "kite/v1", "kite/v2", 1)}, layout)},
		{"the built-in theme's name", zipOf(t, entry{name: "theme.yaml", body: strings.Replace(paperManifest, "name: paper", "name: default", 1)}, layout)},
		{"a name that is a path", zipOf(t, entry{name: "theme.yaml", body: strings.Replace(paperManifest, "name: paper", "name: ../paper", 1)}, layout)},
	} {
		rec := install(t, h, tc.archive, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400\n%s", tc.name, rec.Code, rec.Body.String())
		}
	}
	for _, where := range []string{"themes", "escaped.html"} {
		if _, err := os.Stat(filepath.Join(root, where)); err == nil {
			t.Errorf("%s was written by a refused install", where)
		}
	}
}

func TestRemovingAThemeLeavesTheOneInUse(t *testing.T) {
	root := newProject(t, 1)
	withThemes(t, root)
	h, _ := newWritableServer(t, root)

	for name, want := range map[string]int{
		"default": http.StatusBadRequest, // built in
		"missing": http.StatusNotFound,
		"paper":   http.StatusNoContent,
	} {
		if rec := send(t, h, http.MethodDelete, api.Prefix+"/themes/"+name, nil, nil); rec.Code != want {
			t.Errorf("DELETE %s: status = %d, want %d\n%s", name, rec.Code, want, rec.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "themes", "paper")); err == nil {
		t.Error("paper is still installed")
	}

	write(t, filepath.Join(root, "kite.yaml"), config+"theme:\n  name: kept\n")
	write(t, filepath.Join(root, "themes", "kept", "theme.yaml"), strings.Replace(paperManifest, "name: paper", "name: kept", 1))
	write(t, filepath.Join(root, "themes", "kept", "layouts", "single.html"), "x")
	h, _ = newWritableServer(t, root)
	if rec := send(t, h, http.MethodDelete, api.Prefix+"/themes/kept", nil, nil); rec.Code != http.StatusConflict {
		t.Errorf("removing the theme in use: status = %d, want 409", rec.Code)
	}
}

// A file that belongs to the site rather than one page, as a logo does, is
// kept in static/uploads and named by its path in the site.
func TestAFileOfTheSitesOwnLandsInStaticUploads(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root)

	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("pretend png"))
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, api.Prefix+"/media", &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}
	media := decode[api.Media](t, rec)
	if media.Path != "static/uploads/logo.png" || media.Link != "/uploads/logo.png" || media.URL != "/uploads/logo.png" {
		t.Errorf("media = %+v", media)
	}
	if _, err := os.Stat(filepath.Join(root, "static", "uploads", "logo.png")); err != nil {
		t.Errorf("the file is not in static/uploads: %v", err)
	}
}
