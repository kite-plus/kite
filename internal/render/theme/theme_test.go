package theme_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kite-plus/kite/internal/render/theme"
)

func file(body string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(body)} }

// The lookup chain is part of the frozen contract: a theme author has to be
// able to predict which file wins, and every entry here is a promise.
func TestCandidateOrder(t *testing.T) {
	formats := map[string]theme.Format{
		"html": theme.FormatHTML,
		"rss":  {Name: "rss", Extension: "xml", Infix: "rss"},
	}

	got := theme.Candidates(theme.Target{
		Kind: "single", Type: "post", Layout: "wide", Format: "html", Lang: "zh",
	}, formats)
	want := []string{
		"post/wide.zh.html",
		"post/wide.html",
		"post/single.zh.html",
		"post/single.html",
		"wide.zh.html",
		"wide.html",
		"single.zh.html",
		"single.html",
	}
	if !slices.Equal(got, want) {
		t.Errorf("candidates =\n  %v\nwant\n  %v", got, want)
	}

	rss := theme.Candidates(theme.Target{Kind: "single", Type: "post", Format: "rss"}, formats)
	if !slices.Contains(rss, "post/single.rss.xml") {
		t.Errorf("rss candidates = %v", rss)
	}
}

func TestNoCandidateRecursesThroughSections(t *testing.T) {
	// Section-path recursion is what makes a lookup order impossible to
	// explain, so a deep type name must not expand into a ladder of parents.
	got := theme.Candidates(theme.Target{Kind: "single", Type: "docs/guides/intro"}, nil)
	for _, c := range got {
		if strings.Count(c, "/") > 3 {
			t.Errorf("candidate %q looks like section recursion", c)
		}
	}
	if len(got) > 4 {
		t.Errorf("expected a short chain, got %d candidates: %v", len(got), got)
	}
}

// Precision beats provenance: a site's generic template must not shadow a
// theme's specific one, or installing a theme would silently lose its work.
func TestSiteOverridesThemeAtEqualPrecision(t *testing.T) {
	site := fstest.MapFS{
		"single.html":      file("site generic"),
		"post/single.html": file("site specific"),
	}
	th := fstest.MapFS{
		"single.html":      file("theme generic"),
		"post/single.html": file("theme specific"),
	}
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{
		{Name: "site", FS: site},
		{Name: "theme", FS: th},
	}})

	found, _, ok := e.Lookup(theme.Target{Kind: "single", Type: "post"})
	if !ok {
		t.Fatal("no template found")
	}
	if found.Path != "post/single.html" || found.Source != "site" {
		t.Errorf("got %+v, want site post/single.html", found)
	}
}

func TestThemeSpecificBeatsSiteGeneric(t *testing.T) {
	site := fstest.MapFS{"single.html": file("site generic")}
	th := fstest.MapFS{"post/single.html": file("theme specific")}
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{
		{Name: "site", FS: site},
		{Name: "theme", FS: th},
	}})

	found, _, ok := e.Lookup(theme.Target{Kind: "single", Type: "post"})
	if !ok {
		t.Fatal("no template found")
	}
	if found.Source != "theme" {
		t.Errorf("got %+v, want the theme's more specific template", found)
	}
}

func TestRenderWithBaseAndPartials(t *testing.T) {
	th := fstest.MapFS{
		"baseof.html":         file(`<html><body>{{ block "main" . }}{{ end }}{{ partial "foot.html" . }}</body></html>`),
		"single.html":         file(`{{ define "main" }}<h1>{{ .Title }}</h1>{{ end }}`),
		"_partials/foot.html": file(`<footer>{{ .Footer }}</footer>`),
	}
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{{Name: "theme", FS: th}}})

	out, err := e.Render(theme.Target{Kind: "single"}, map[string]any{"Title": "Hi", "Footer": "bye"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := string(out)
	for _, want := range []string{"<h1>Hi</h1>", "<footer>bye</footer>"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %s", want, got)
		}
	}
}

func TestSitePartialOverridesThemePartial(t *testing.T) {
	site := fstest.MapFS{"_partials/foot.html": file(`<footer>site</footer>`)}
	th := fstest.MapFS{
		"single.html":         file(`{{ partial "foot.html" . }}`),
		"_partials/foot.html": file(`<footer>theme</footer>`),
	}
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{
		{Name: "site", FS: site},
		{Name: "theme", FS: th},
	}})

	out, err := e.Render(theme.Target{Kind: "single"}, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "site") {
		t.Errorf("the site partial did not win: %s", out)
	}
}

// Namespaced helpers are the whole point of the function contract, so the
// {{ ns.Method }} form has to keep working.
func TestNamespacedFunctions(t *testing.T) {
	cases := map[string]string{
		`{{ str.Upper "abc" }}`:               "ABC",
		`{{ str.Title "hello world" }}`:       "Hello World",
		`{{ str.Truncate 5 "abcdefgh" }}`:     "abcde…",
		`{{ math.Add 2 3 }}`:                  "5",
		`{{ url.Anchorize "Web Dev!" }}`:      "web-dev",
		`{{ time.Format "2006" .When }}`:      "2026",
		`{{ time.Minutes .Dur }}`:             "3",
		`{{ collections.Len (slice 1 2 3) }}`: "3",
		`{{ default "fallback" "" }}`:         "fallback",
		`{{ (dict "k" "v").k }}`:              "v",
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			th := fstest.MapFS{"single.html": file(src)}
			e := theme.NewEngine(theme.Options{Sources: []theme.Source{{Name: "theme", FS: th}}})
			out, err := e.Render(theme.Target{Kind: "single"}, map[string]any{
				"When": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				"Dur":  150 * time.Second,
			})
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if strings.TrimSpace(string(out)) != want {
				t.Errorf("got %q, want %q", out, want)
			}
		})
	}
}

// A template that could read the clock would be a hidden build input.
func TestThereIsNoTimeNow(t *testing.T) {
	th := fstest.MapFS{"single.html": file(`{{ time.Now }}`)}
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{{Name: "theme", FS: th}}})
	if _, err := e.Render(theme.Target{Kind: "single"}, nil); err == nil {
		t.Fatal("time.Now must not exist: templates read .Site.BuildTime instead")
	}
}

func TestMissingTemplateNamesTheChain(t *testing.T) {
	e := theme.NewEngine(theme.Options{Sources: []theme.Source{{Name: "theme", FS: fstest.MapFS{}}}})
	_, err := e.Render(theme.Target{Kind: "single", Type: "post"}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "post/single.html") {
		t.Errorf("the error should list what was tried: %v", err)
	}
}

func TestManifestRejectsUnknownAPIVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"theme.yaml":          file("name: x\nversion: 1.0.0\napiVersion: kite/v99\n"),
		"layouts/single.html": file("x"),
	}
	_, err := theme.Load(fsys)
	if err == nil {
		t.Fatal("a theme built for another contract version must be refused, not half rendered")
	}
	if !strings.Contains(err.Error(), "kite/v99") {
		t.Errorf("the error should name the declared version: %v", err)
	}
}

func TestManifestSettingsDrivenDefaults(t *testing.T) {
	fsys := fstest.MapFS{
		"theme.yaml": file(`name: x
version: 1.0.0
apiVersion: kite/v1
settings:
  - key: accent
    type: color
    default: "#000"
  - key: show_toc
    type: boolean
    default: true
`),
		"layouts/single.html": file("x"),
	}
	th, err := theme.Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defaults := th.Manifest.DefaultSettings()
	if defaults["accent"] != "#000" || defaults["show_toc"] != true {
		t.Errorf("defaults = %v", defaults)
	}
	if !th.Manifest.SupportsStatic() || !th.Manifest.SupportsDynamic() {
		t.Error("a theme that declares no capabilities supports both runtimes")
	}
}

func TestReservedDirectories(t *testing.T) {
	for path, want := range map[string]bool{
		"_partials/head.html": true,
		"post/single.html":    false,
		"_markup/x.html":      true,
		"single.html":         false,
	} {
		if got := theme.IsReserved(path); got != want {
			t.Errorf("IsReserved(%q) = %v, want %v", path, got, want)
		}
	}
}

// A theme offers layouts by name, and each has to be a template's file name
// with a template behind it for every type it is offered to: the admin lists
// them, and a layout that quietly fell back would be a broken promise.
func TestLayoutsAreNamesWithTemplatesBehindThem(t *testing.T) {
	manifest := func(layouts string) string {
		return "name: x\nversion: 1.0.0\napiVersion: kite/v1\nlayouts:\n" + layouts
	}
	for _, tc := range []struct {
		name    string
		fsys    fstest.MapFS
		refused string // part of the error, or empty when the theme loads
	}{
		{
			name: "offered to pages, drawn from the page directory",
			fsys: fstest.MapFS{
				"theme.yaml":              file(manifest("  - {name: links, label: Links, types: [page]}\n")),
				"layouts/single.html":     file("x"),
				"layouts/page/links.html": file("x"),
			},
		},
		{
			name: "offered to every type, drawn from the top",
			fsys: fstest.MapFS{
				"theme.yaml":          file(manifest("  - {name: plain}\n")),
				"layouts/single.html": file("x"),
				"layouts/plain.html":  file("x"),
			},
		},
		{
			name: "offered to pages with no template",
			fsys: fstest.MapFS{
				"theme.yaml":          file(manifest("  - {name: links, types: [page]}\n")),
				"layouts/single.html": file("x"),
			},
			refused: "page/links.html",
		},
		{
			name: "offered to every type but drawn for pages only",
			fsys: fstest.MapFS{
				"theme.yaml":              file(manifest("  - {name: links}\n")),
				"layouts/single.html":     file("x"),
				"layouts/page/links.html": file("x"),
			},
			refused: "every type",
		},
		{
			name: "a name that is a path",
			fsys: fstest.MapFS{
				"theme.yaml":          file(manifest("  - {name: ../single}\n")),
				"layouts/single.html": file("x"),
			},
			refused: "../single",
		},
		{
			name: "one name declared twice",
			fsys: fstest.MapFS{
				"theme.yaml":         file(manifest("  - {name: plain}\n  - {name: plain}\n")),
				"layouts/plain.html": file("x"),
			},
			refused: "twice",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := theme.Load(tc.fsys)
			switch {
			case tc.refused == "" && err != nil:
				t.Fatalf("refused: %v", err)
			case tc.refused != "" && err == nil:
				t.Fatalf("loaded, want a refusal naming %q", tc.refused)
			case tc.refused != "" && !strings.Contains(err.Error(), tc.refused):
				t.Errorf("the refusal does not name %q: %v", tc.refused, err)
			}
		})
	}
}
