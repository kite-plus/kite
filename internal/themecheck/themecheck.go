// Package themecheck proves that a theme draws the same pages in both
// runtimes.
//
// The contract a theme is written against is this check rather than the
// documentation of it (docs/design/theme-system.md §9.4). A small site that
// uses every kind of page is built with the theme, every file the build
// writes is then asked of a server, and every byte is compared. A theme that
// passes can be previewed with `kite run` and trusted to publish what the
// preview showed.
//
// The site is published under a path, as a GitHub Pages project site is, and
// a link a page makes to the root of the host is reported as well: it works
// on a site at the root and breaks on every other one.
package themecheck

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/serve"
	"github.com/kite-plus/kite/internal/site"
)

//go:embed all:fixture
var fixture embed.FS

// installed is the name the theme is given inside the fixture site. The name
// "default" would pick the theme built into Kite instead of the one checked.
const installed = "under-check"

// At is the instant the fixture renders at, so that its scheduled post falls
// on the same side of it on every run.
var At = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// Report is what a check found.
type Report struct {
	Theme    string       `json:"theme"`
	Compared int          `json:"compared"`
	Differ   []Difference `json:"differ,omitempty"`

	// Outside holds each link that leaves the site for the root of its host,
	// with the first page that makes it.
	Outside []Difference `json:"outside,omitempty"`
}

// Difference is a file the two runtimes did not produce alike, or a page
// that links outside the site.
type Difference struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

// OK reports whether every file matched and every link stayed in the site.
func (r *Report) OK() bool {
	return r.Compared > 0 && len(r.Differ) == 0 && len(r.Outside) == 0
}

// rootLink matches a link that starts at the root of its host, in the
// attributes that hold one.
var rootLink = regexp.MustCompile(`\b(?:href|src|action|poster)\s*=\s*["'](/[^"'\s]*)`)

// Check renders the fixture site with a theme, built and served, and
// compares every file.
func Check(ctx context.Context, fsys fs.FS) (*Report, error) {
	root, err := os.MkdirTemp("", "kite-theme-check-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(root) }()

	files, err := fs.Sub(fixture, "fixture")
	if err != nil {
		return nil, err
	}
	if err := os.CopyFS(root, files); err != nil {
		return nil, fmt.Errorf("themecheck: write the fixture site: %w", err)
	}
	if err := os.CopyFS(filepath.Join(root, "themes", installed), fsys); err != nil {
		return nil, fmt.Errorf("themecheck: install the theme: %w", err)
	}
	if err := addLayoutPages(root, fsys); err != nil {
		return nil, err
	}
	config, err := os.OpenFile(filepath.Join(root, "kite.yaml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(config, "theme:\n  name: %s\n", installed)
	if cerr := config.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}

	s, err := site.Open(ctx, root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = s.Close() }()

	out := filepath.Join(root, "public")
	if _, _, err := s.Build(ctx, site.BuildOptions{OutDir: out, Now: At}); err != nil {
		return nil, fmt.Errorf("themecheck: build: %w", err)
	}
	srv, err := serve.NewWithClock(ctx, s, serve.Options{}, func() time.Time { return At })
	if err != nil {
		return nil, fmt.Errorf("themecheck: serve: %w", err)
	}
	handler := srv.Handler()

	report := &Report{Theme: s.Theme.Manifest.Name}
	seen := make(map[string]bool)
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(out, p)
		if err != nil {
			return err
		}
		built, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		url := s.Resolver.Rel(filepath.ToSlash(rel))
		if strings.HasSuffix(rel, ".html") {
			for _, m := range rootLink.FindAllStringSubmatch(string(built), -1) {
				link := m[1]
				// A protocol-relative address names a host of its own.
				if strings.HasPrefix(link, "//") || seen[link] {
					continue
				}
				path := link
				if i := strings.IndexAny(path, "?#"); i >= 0 {
					path = path[:i]
				}
				if _, inside := s.Resolver.SitePath(path); !inside {
					seen[link] = true
					report.Outside = append(report.Outside, Difference{URL: url, Detail: "links to " + link})
				}
			}
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, url, nil))

		want := http.StatusOK
		if rel == "404.html" {
			want = http.StatusNotFound
		}
		switch {
		case rec.Code != want:
			report.Differ = append(report.Differ, Difference{URL: url,
				Detail: fmt.Sprintf("served with status %d, want %d", rec.Code, want)})
		case rec.Body.String() != string(built):
			report.Differ = append(report.Differ, Difference{URL: url,
				Detail: firstDifference(string(built), rec.Body.String())})
		}
		report.Compared++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

// firstDifference points at the first line that differs, which is where a
// theme author starts looking.
func firstDifference(built, served string) string {
	b, s := strings.Split(built, "\n"), strings.Split(served, "\n")
	for i := range max(len(b), len(s)) {
		var x, y string
		if i < len(b) {
			x = b[i]
		}
		if i < len(s) {
			y = s[i]
		}
		if x != y {
			return fmt.Sprintf("line %d: built %q, served %q", i+1, x, y)
		}
	}
	return "the same line by line, but not the same bytes"
}

// addLayoutPages gives the fixture one item for every layout the theme
// offers, of each type it is offered to, so each layout is drawn and
// compared like the rest of the site rather than only found to exist. A
// layout offered to every type is drawn as a page.
func addLayoutPages(root string, fsys fs.FS) error {
	th, err := theme.Load(fsys)
	if err != nil {
		return err
	}
	n := 0
	for _, l := range th.Manifest.Layouts {
		kinds := l.Types
		if len(kinds) == 0 {
			kinds = []string{"page"}
		}
		for _, kind := range kinds {
			var rel string
			switch kind {
			case "page":
				rel = filepath.Join("content", "pages", "layout-"+l.Name+".md")
			case "post":
				rel = filepath.Join("content", "posts", "layout-"+l.Name, "index.md")
			default:
				continue // the fixture has posts and pages only
			}
			n++
			body := fmt.Sprintf("---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ4%02d\ntitle: Layout %s\nslug: layout-%s\n"+
				"status: published\npublished_at: 2026-01-01T00:00:00Z\nlayout: %s\n---\n\n"+
				"- [Kite](https://example.com/kite/) A link, as a list of them is written.\n"+
				"- [Explore](https://example.com/explore/) Another one.\n", n, l.Name, l.Name, l.Name)
			path := filepath.Join(root, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return fmt.Errorf("themecheck: write a page for layout %s: %w", l.Name, err)
			}
		}
	}
	return nil
}
