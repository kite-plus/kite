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
	for _, d := range report.Outside {
		t.Errorf("%s %s, outside the site", d.URL, d.Detail)
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
	theme := defaultThemeWith(t, "layouts/single.html", `{{ define "main" }}`,
		`{{ define "main" }}{{ with .Request }}<p>served from {{ .Path }}</p>{{ end }}`)

	report, err := themecheck.Check(t.Context(), theme)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if report.OK() {
		t.Fatal("a theme that draws the request passed")
	}
	var named bool
	for _, d := range report.Differ {
		if strings.HasPrefix(d.URL, "/blog/posts/") && strings.Contains(d.Detail, "served from") {
			named = true
		}
		if !strings.HasPrefix(d.URL, "/blog/posts/") && d.URL != "/blog/about/index.html" {
			t.Errorf("%s differs, but only single pages draw the request: %s", d.URL, d.Detail)
		}
	}
	if !named {
		t.Errorf("no difference points at the line that changed: %+v", report.Differ)
	}
}

// A theme that writes the root of the host into a link draws the same page in
// both runtimes, and still breaks on every site published under a path.
func TestAThemeThatLinksToTheRootOfTheHostIsCaught(t *testing.T) {
	theme := defaultThemeWith(t, "layouts/_partials/footer.html",
		`href="{{ url.Rel "rss.xml" }}"`, `href="/rss.xml"`)

	report, err := themecheck.Check(t.Context(), theme)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if report.OK() {
		t.Fatal("a theme that links to /rss.xml passed")
	}
	if len(report.Differ) != 0 {
		t.Errorf("both runtimes draw the same link, yet %d files differ", len(report.Differ))
	}
	// Named once, however many pages carry the footer.
	if len(report.Outside) != 1 || report.Outside[0].Detail != "links to /rss.xml" {
		t.Errorf("outside = %+v, want the one link to /rss.xml", report.Outside)
	}
}

// defaultThemeWith is the built-in theme with one replacement made in one of
// its files.
func defaultThemeWith(t *testing.T, file, old, replacement string) fstest.MapFS {
	t.Helper()
	theme := fstest.MapFS{}
	err := fs.WalkDir(themes.Default(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(themes.Default(), p)
		if err != nil {
			return err
		}
		if p == file {
			changed := strings.Replace(string(data), old, replacement, 1)
			if changed == string(data) {
				t.Fatalf("%s does not contain %q", file, old)
			}
			data = []byte(changed)
		}
		theme[p] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return theme
}
