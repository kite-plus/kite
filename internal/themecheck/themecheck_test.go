package themecheck_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kite-plus/kite/internal/themecheck"
	"github.com/kite-plus/kite/themes"
)

// The theme Kite ships with is the first one the contract has to hold for.
func TestTheBuiltInThemeDrawsTheSamePagesBuiltAndServed(t *testing.T) {
	report, err := themecheck.Check(t.Context(), themes.Default())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, d := range report.Differ {
		t.Errorf("%s: %s", d.URL, d.Detail)
	}
	// Every kind of page: seven posts and a page, three pages of the home
	// listing and of the posts, the pages listing, both taxonomies, their
	// five terms and the 404. Then the feed, the sitemap and the image a
	// page bundle carries.
	if want := 8 + 3 + 3 + 1 + 2 + 5 + 1 + 3; report.Compared != want {
		t.Errorf("%d files compared, want %d; the fixture no longer covers what it did", report.Compared, want)
	}
	if !report.OK() {
		t.Error("the report is not OK")
	}
}

// A theme that draws something only a server knows cannot publish what its
// preview showed, and the check has to say where.
func TestAThemeThatDependsOnTheRuntimeIsCaught(t *testing.T) {
	theme := fstest.MapFS{}
	err := fs.WalkDir(themes.Default(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(themes.Default(), p)
		if err != nil {
			return err
		}
		if p == "layouts/single.html" {
			data = []byte(strings.Replace(string(data), `{{ define "main" }}`,
				`{{ define "main" }}{{ with .Request }}<p>served from {{ .Path }}</p>{{ end }}`, 1))
		}
		theme[p] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	report, err := themecheck.Check(t.Context(), theme)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if report.OK() {
		t.Fatal("a theme that draws the request passed")
	}
	var named bool
	for _, d := range report.Differ {
		if strings.HasPrefix(d.URL, "/posts/") && strings.Contains(d.Detail, "served from") {
			named = true
		}
		if !strings.HasPrefix(d.URL, "/posts/") && d.URL != "/about/index.html" {
			t.Errorf("%s differs, but only single pages draw the request: %s", d.URL, d.Detail)
		}
	}
	if !named {
		t.Errorf("no difference points at the line that changed: %+v", report.Differ)
	}
}
