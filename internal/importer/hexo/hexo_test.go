package hexo_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/importer/hexo"
	"github.com/kite-plus/kite/internal/project"
)

func newProject(t *testing.T) *project.Project {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "kite.yaml"), []byte("site:\n  title: T\n  baseURL: https://example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := project.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func items(t *testing.T, p *project.Project) map[string]*content.Content {
	t.Helper()
	scan, err := p.Scanner().Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(scan.Problems) > 0 {
		t.Fatalf("the imported files do not read: %v", scan.Problems)
	}
	out := map[string]*content.Content{}
	for _, e := range scan.Entries {
		out[string(e.Item.Kind)+":"+e.Item.Slug] = e.Item
	}
	return out
}

func TestTheConfigSaysWhereTheSiteLives(t *testing.T) {
	cfg, err := hexo.ReadConfig("testdata/site")
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if cfg.Title != "旧博客" || cfg.Description != "Notes on things" || cfg.Author != "Amy" ||
		cfg.Language != "zh-CN" || cfg.Timezone != "Asia/Shanghai" || cfg.URL != "https://example.com/blog" {
		t.Errorf("config = %+v", cfg)
	}

	// Before Hexo 5 the path lived in root alone.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "_config.yml"), []byte("url: https://example.com/\nroot: /notes/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg, err := hexo.ReadConfig(dir); err != nil || cfg.URL != "https://example.com/notes" || cfg.Permalink != ":year/:month/:day/:title/" {
		t.Errorf("config = %+v, %v", cfg, err)
	}
	if _, err := hexo.ReadConfig(t.TempDir()); err == nil || !strings.Contains(err.Error(), "_config.yml") {
		t.Errorf("a folder with no _config.yml: %v", err)
	}
}

// A Hexo site's posts, drafts and pages become items that read as Kite's own,
// the files beside them come along, and every address Hexo published one at
// keeps leading to it.
func TestAHexoSiteBecomesKiteContent(t *testing.T) {
	p := newProject(t)
	report, err := hexo.Import(t.Context(), "testdata/site", p)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if report.Posts != 2 || report.Drafts != 1 || report.Pages != 2 || report.Files != 4 || report.Aliases != 3 {
		t.Errorf("report = %+v", report)
	}
	if !slices.Equal(report.Tags, []string{"content/posts/hello-world"}) {
		t.Errorf("tags left in = %v", report.Tags)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Path != "source/_posts/loose.png" {
		t.Errorf("skipped = %+v", report.Skipped)
	}

	got := items(t, p)
	hello := got["post:hello-world"]
	if hello == nil {
		t.Fatalf("items = %v", got)
	}
	if hello.Title != "Hello World" || hello.Status != content.StatusPublished {
		t.Errorf("hello = %q %q", hello.Title, hello.Status)
	}
	// Written without a zone, the dates were Shanghai's.
	if want := time.Date(2026, 1, 2, 0, 30, 0, 0, time.UTC); hello.PublishedAt == nil || !hello.PublishedAt.Equal(want) {
		t.Errorf("published = %v, want %v", hello.PublishedAt, want)
	}
	if want := time.Date(2026, 1, 5, 2, 0, 0, 0, time.UTC); !hello.UpdatedAt.Equal(want) {
		t.Errorf("updated = %v, want %v", hello.UpdatedAt, want)
	}
	if !slices.Equal(hello.Terms("tags"), []string{"hexo", "notes"}) ||
		!slices.Equal(hello.Terms("categories"), []string{"Life", "Diary", "Tech"}) {
		t.Errorf("terms = %v", hello.Taxonomies)
	}
	if _, kept := hello.Meta["layout"]; kept || hello.Meta["comments"] != false {
		t.Errorf("meta = %v", hello.Meta)
	}
	if !slices.Equal(hello.Aliases, []string{"/2026/01/02/hello-world/"}) {
		t.Errorf("aliases = %v", hello.Aliases)
	}
	for _, want := range []string{"The opening words.", "<!-- more -->", "![A picture](a.png)", "{% note info %}"} {
		if !strings.Contains(hello.Body.Raw, want) {
			t.Errorf("body lacks %q:\n%s", want, hello.Body.Raw)
		}
	}
	if _, err := os.Stat(filepath.Join(p.Root, "content", "posts", "hello-world", "a.png")); err != nil {
		t.Errorf("the post's own file did not come into its bundle: %v", err)
	}

	old := got["post:old-one"]
	if old == nil {
		t.Fatalf("items = %v", got)
	}
	if old.Title != "An old one: from 2019" || !slices.Equal(old.Terms("tags"), []string{"go"}) {
		t.Errorf("old = %q %v", old.Title, old.Taxonomies)
	}
	if old.Meta["description"] != "Written a while ago." || old.Meta["cover"] != "/images/old.png" {
		t.Errorf("old meta = %v", old.Meta)
	}
	// 23:30 in UTC was the next day in Shanghai, where Hexo dated its address.
	if !slices.Equal(old.Aliases, []string{"/2019/05/02/2019/old-one/"}) {
		t.Errorf("old aliases = %v", old.Aliases)
	}

	if idea := got["post:idea"]; idea == nil || idea.Status != content.StatusDraft || idea.PublishedAt != nil {
		t.Errorf("draft = %+v", idea)
	}

	about := got["page:about"]
	if about == nil || len(about.Aliases) != 0 {
		t.Fatalf("about = %+v", about)
	}
	if _, kept := about.Meta["layout"]; kept {
		t.Errorf("about keeps layout: page: %v", about.Meta)
	}
	links := got["page:links"]
	if links == nil || links.Meta["layout"] != "links" || !slices.Equal(links.Aliases, []string{"/links.html"}) {
		t.Errorf("links = %+v", links)
	}

	for _, f := range []string{"static/about/me.jpg", "static/images/old.png", "static/CNAME"} {
		if _, err := os.Stat(filepath.Join(p.Root, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s was not copied: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(p.Root, "static", "_data")); err == nil {
		t.Error("a folder Hexo does not publish was copied")
	}
}

// Importing into a project that already has content takes nothing from it: a
// slug in use gets a number, and a file in the way is left where it is.
func TestAnImportTakesNothingFromTheProject(t *testing.T) {
	p := newProject(t)
	if _, err := hexo.Import(t.Context(), "testdata/site", p); err != nil {
		t.Fatal(err)
	}
	report, err := hexo.Import(t.Context(), "testdata/site", p)
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	got := items(t, p)
	if got["post:hello-world"] == nil || got["post:hello-world-2"] == nil || got["page:about-2"] == nil {
		t.Errorf("items = %v", got)
	}
	if len(got["page:about-2"].Aliases) != 1 || got["page:about-2"].Aliases[0] != "/about/" {
		t.Errorf("about-2 aliases = %v", got["page:about-2"].Aliases)
	}
	if report.Files != 1 {
		t.Errorf("files = %d, want only the new bundle's file", report.Files)
	}
}
