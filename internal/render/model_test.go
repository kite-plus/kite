package render_test

import (
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/markdown"
)

func page(doc *markdown.Document, opts render.PageOptions) render.Page {
	opts.Rendered = doc
	return render.NewPage(&content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Title: "T", Slug: "t"}, opts)
}

func TestReadingTimeUsesAPacePerScript(t *testing.T) {
	for _, tc := range []struct {
		name       string
		words, cjk int
		want       time.Duration
	}{
		{"empty", 0, 0, time.Minute},
		{"spaced words", 2200, 0, 10 * time.Minute},
		{"cjk characters", 2000, 2000, 5 * time.Minute},
		{"both", 2220, 2000, 6 * time.Minute},
		{"rounds up", 221, 0, 2 * time.Minute},
	} {
		p := page(&markdown.Document{WordCount: tc.words, CJKCount: tc.cjk}, render.PageOptions{})
		if got := p.ReadingTime(); got != tc.want {
			t.Errorf("%s: ReadingTime = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A theme reaches a neighbor through {{ with }}, which only skips a missing
// one if it is an untyped nil.
func TestMissingNeighborsAreSkippedByWith(t *testing.T) {
	const src = `{{ with .Prev }}prev={{ .Title }}{{ end }} {{ with .Next }}next={{ .Title }}{{ end }}`
	tmpl := template.Must(template.New("t").Parse(src))

	older := page(&markdown.Document{}, render.PageOptions{})
	for _, tc := range []struct {
		name string
		opts render.PageOptions
		want string
	}{
		{"alone", render.PageOptions{}, ""},
		{"newest", render.PageOptions{Prev: older}, "prev=T"},
		{"oldest", render.PageOptions{Next: older}, "next=T"},
	} {
		var b strings.Builder
		if err := tmpl.Execute(&b, page(&markdown.Document{}, tc.opts)); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := strings.TrimSpace(b.String()); got != tc.want {
			t.Errorf("%s: rendered %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A post published just after midnight in Shanghai is stored in UTC, the day
// before. In a site set to that zone, every date a template sees is shown
// there, so the post reads as written on the day it was.
func TestDatesAreShownInTheSitesZone(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	stored := time.Date(2026, 9, 25, 16, 30, 0, 0, time.UTC)
	item := &content.Content{
		ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Title: "T", Slug: "t",
		CreatedAt: stored, UpdatedAt: stored, PublishedAt: &stored,
	}

	shown := render.NewPage(item, render.PageOptions{Location: shanghai})
	for name, got := range map[string]time.Time{
		"Date": shown.Date(), "PublishDate": shown.PublishDate(), "Lastmod": shown.Lastmod(),
	} {
		if got.Day() != 26 || got.Location() != shanghai || !got.Equal(stored) {
			t.Errorf("%s = %v, want the same instant on the 26th in Shanghai", name, got)
		}
	}
	site := render.NewSite(render.SiteInfo{BuildTime: stored, Location: shanghai})
	if got := site.BuildTime(); got.Day() != 26 {
		t.Errorf("BuildTime = %v, want the 26th in Shanghai", got)
	}

	// With no zone named, a date is shown as it was written.
	if got := render.NewPage(item, render.PageOptions{}).PublishDate(); got.Day() != 25 || got.Location() != time.UTC {
		t.Errorf("with no zone, PublishDate = %v, want it as stored", got)
	}
}
