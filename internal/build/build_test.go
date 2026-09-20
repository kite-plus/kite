package build_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/hook/builtin"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/reader"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/markdown"
	"github.com/kite-plus/kite/internal/render/theme"
	kurl "github.com/kite-plus/kite/internal/render/url"
	"github.com/kite-plus/kite/themes"
)

type fixture struct {
	root    string
	out     string
	reader  *reader.Reader
	types   *content.Registry
	hooks   *hook.Bus
	resolve *kurl.Resolver
	engine  *theme.Engine
}

func newFixture(t *testing.T, posts int) *fixture {
	t.Helper()
	root := t.TempDir()

	for i := range posts {
		body := fmt.Sprintf(`---
id: 01J8KQ2P3R4S5T6V7W8X9YZ%03d
title: Post %02d
slug: post-%02d
status: published
published_at: 2026-01-%02dT00:00:00Z
tags: [Go]
---

# Heading

Body of post %02d.
`, i, i, i, i+1, i)
		p := filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i), "index.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	types := content.DefaultRegistry()
	ix, err := index.Open(root, types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}

	resolver, err := kurl.New(kurl.Options{
		BaseURL: "https://example.com", Style: kurl.StyleDirectory,
		PaginationPath: "page", TaxonomyRoute: "/:taxonomy", TermRoute: "/:taxonomy/:term",
	}, types)
	if err != nil {
		t.Fatal(err)
	}

	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}

	bus := hook.NewBus()
	builtin.Register(bus, builtin.DefaultOptions())

	return &fixture{
		root:    root,
		out:     filepath.Join(root, "public"),
		reader:  reader.New(ix.DB()),
		types:   types,
		hooks:   bus,
		resolve: resolver,
		engine:  theme.NewEngine(theme.Options{Sources: []theme.Source{{Name: "default", FS: th.Layouts}}}),
	}
}

func (f *fixture) run(t *testing.T, outDir string, mutate func(*build.Options)) (build.Stats, []string) {
	t.Helper()
	emitter, err := build.NewEmitter(outDir)
	if err != nil {
		t.Fatal(err)
	}
	opts := build.Options{
		Site: render.SiteInfo{
			Title: "Test", BaseURL: "https://example.com", Language: "en",
		},
		Reader:   f.reader,
		Resolver: f.resolve,
		Engine:   f.engine,
		Markdown: markdown.New(markdown.DefaultOptions()),
		Hooks:    f.hooks,
		Types:    f.types,
		Emitter:  emitter,
		PageSize: 3,
		Now:      time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}
	if mutate != nil {
		mutate(&opts)
	}
	b, err := build.New(opts)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := b.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return stats, emitter.Files()
}

func TestBuildProducesEveryPageKind(t *testing.T) {
	f := newFixture(t, 5)
	_, files := f.run(t, f.out, nil)

	for _, want := range []string{
		"index.html",        // home
		"page/2/index.html", // home pagination
		"posts/index.html",  // listing
		"posts/post-00/index.html",
		"tags/index.html",    // taxonomy
		"tags/go/index.html", // term
		"404.html",
		"sitemap.xml",
		"rss.xml",
	} {
		if !slices.Contains(files, want) {
			t.Errorf("missing %s\ngot: %v", want, files)
		}
	}
}

// Two builds of the same input must produce identical bytes. Without this the
// recorded dependencies, and any cache built on them, mean nothing.
func TestBuildIsReproducible(t *testing.T) {
	f := newFixture(t, 8)

	_, first := f.run(t, f.out, nil)
	firstBytes := readAll(t, f.out, first)

	second := filepath.Join(f.root, "public2")
	_, files := f.run(t, second, nil)
	secondBytes := readAll(t, second, files)

	if len(firstBytes) != len(secondBytes) {
		t.Fatalf("file counts differ: %d vs %d", len(firstBytes), len(secondBytes))
	}
	for path, data := range firstBytes {
		if secondBytes[path] != data {
			t.Errorf("%s differs between two builds", path)
		}
	}
}

// Output must not depend on map iteration order anywhere in the pipeline.
func TestBuildOutputIsStableAcrossRuns(t *testing.T) {
	f := newFixture(t, 6)
	_, base := f.run(t, f.out, nil)

	for i := range 3 {
		dir := filepath.Join(f.root, fmt.Sprintf("run%d", i))
		_, files := f.run(t, dir, nil)
		if !slices.Equal(files, base) {
			t.Fatalf("run %d produced a different file set:\n %v\nvs\n %v", i, files, base)
		}
	}
}

func TestPaginationSplitsListings(t *testing.T) {
	f := newFixture(t, 7) // page size 3 gives 3 pages
	_, files := f.run(t, f.out, nil)

	for _, want := range []string{"index.html", "page/2/index.html", "page/3/index.html"} {
		if !slices.Contains(files, want) {
			t.Errorf("missing %s", want)
		}
	}
	if slices.Contains(files, "page/4/index.html") {
		t.Error("produced a page beyond the last one")
	}

	home := readFile(t, f.out, "index.html")
	if !strings.Contains(home, "1 / 3") {
		t.Errorf("home page does not show its position: %s", excerptOf(home, "pagination"))
	}
}

func TestDraftsAreExcludedUnlessRequested(t *testing.T) {
	f := newFixture(t, 3)
	draft := filepath.Join(f.root, "content", "posts", "hidden", "index.md")
	if err := os.MkdirAll(filepath.Dir(draft), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ900\ntitle: Hidden\nslug: hidden\nstatus: draft\n---\n\nsecret\n"
	if err := os.WriteFile(draft, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	ix, err := index.Open(f.root, f.types)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.reader = reader.New(ix.DB())

	_, files := f.run(t, f.out, nil)
	if slices.Contains(files, "posts/hidden/index.html") {
		t.Error("a draft was published")
	}

	_, withDrafts := f.run(t, filepath.Join(f.root, "drafts"), func(o *build.Options) { o.IncludeDrafts = true })
	if !slices.Contains(withDrafts, "posts/hidden/index.html") {
		t.Error("--drafts did not include the draft")
	}
}

// A failed build must leave the previous site in place rather than a mixture.
func TestFailedBuildLeavesPreviousOutputIntact(t *testing.T) {
	f := newFixture(t, 2)
	f.run(t, f.out, nil)
	before := readFile(t, f.out, "index.html")

	broken := theme.NewEngine(theme.Options{Sources: []theme.Source{
		{Name: "broken", FS: os.DirFS(t.TempDir())},
	}})
	emitter, err := build.NewEmitter(f.out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := build.New(build.Options{
		Site: render.SiteInfo{Title: "Test"}, Reader: f.reader, Resolver: f.resolve,
		Engine: broken, Hooks: f.hooks, Types: f.types, Emitter: emitter, PageSize: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Run(context.Background()); err == nil {
		t.Fatal("expected the build to fail with no templates")
	}
	emitter.Discard()

	if got := readFile(t, f.out, "index.html"); got != before {
		t.Error("a failed build damaged the previously published site")
	}
}

func TestEmitterRefusesToEscapeOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	e, err := build.NewEmitter(filepath.Join(dir, "public"))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Write("../escaped.html", []byte("x")); err == nil {
		t.Fatal("expected a refusal for a path outside the output directory")
	}
}

func TestSitemapAndFeedComeFromHooks(t *testing.T) {
	f := newFixture(t, 3)

	// With no hooks registered there is no sitemap and no feed: they are
	// extensions, not hard-wired build steps.
	bare := hook.NewBus()
	_, files := f.run(t, filepath.Join(f.root, "bare"), func(o *build.Options) { o.Hooks = bare })
	for _, unwanted := range []string{"sitemap.xml", "rss.xml"} {
		if slices.Contains(files, unwanted) {
			t.Errorf("%s was produced without the hook that owns it", unwanted)
		}
	}

	_, withHooks := f.run(t, f.out, nil)
	for _, want := range []string{"sitemap.xml", "rss.xml"} {
		if !slices.Contains(withHooks, want) {
			t.Errorf("missing %s", want)
		}
	}

	sitemap := readFile(t, f.out, "sitemap.xml")
	if strings.Contains(sitemap, "/404") {
		t.Error("the error page must not appear in the sitemap")
	}
}

func readAll(t *testing.T, root string, files []string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(files))
	for _, f := range files {
		out[f] = readFile(t, root, f)
	}
	return out
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func excerptOf(s, marker string) string {
	i := strings.Index(s, marker)
	if i < 0 {
		return s[:min(200, len(s))]
	}
	return s[i:min(i+200, len(s))]
}
