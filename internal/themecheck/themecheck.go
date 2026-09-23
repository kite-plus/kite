// Package themecheck proves that a theme draws the same pages in both
// runtimes.
//
// The contract a theme is written against is this check rather than the
// documentation of it (docs/design/theme-system.md §9.4). A small site that
// uses every kind of page is built with the theme, every file the build
// writes is then asked of a server, and every byte is compared. A theme that
// passes can be previewed with `kite run` and trusted to publish what the
// preview showed.
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
	"strings"
	"time"

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
}

// Difference is a file the two runtimes did not produce alike.
type Difference struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

// OK reports whether every file matched.
func (r *Report) OK() bool { return r.Compared > 0 && len(r.Differ) == 0 }

// Check renders the fixture site with a theme, built and served, and
// compares every file.
func Check(ctx context.Context, theme fs.FS) (*Report, error) {
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
	if err := os.CopyFS(filepath.Join(root, "themes", installed), theme); err != nil {
		return nil, fmt.Errorf("themecheck: install the theme: %w", err)
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
		url := "/" + filepath.ToSlash(rel)
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
