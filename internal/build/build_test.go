package build_test

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
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

// fixtureNow is the build clock unless a test sets its own.
var fixtureNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func (f *fixture) builder(t *testing.T, emitter *build.Emitter, mutate func(*build.Options)) *build.Builder {
	t.Helper()
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
		Now:      fixtureNow,
	}
	if mutate != nil {
		mutate(&opts)
	}
	b, err := build.New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// add writes a file into the project and indexes it.
func (f *fixture) add(t *testing.T, rel, body string) {
	t.Helper()
	p := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := index.Open(f.root, f.types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Close() })
	if _, err := ix.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.reader = reader.New(ix.DB())
}

func (f *fixture) run(t *testing.T, outDir string, mutate func(*build.Options)) (build.Stats, []string) {
	t.Helper()
	emitter, err := build.NewEmitter(outDir)
	if err != nil {
		t.Fatal(err)
	}
	b := f.builder(t, emitter, mutate)
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

// A post links to the one published before it and the one after, and a page
// stays out of that run: the neighbors of an essay are essays.
func TestSinglePagesKnowTheirNeighbors(t *testing.T) {
	f := newFixture(t, 3) // post-02 is newest, post-00 oldest

	about := filepath.Join(f.root, "content", "pages", "about.md")
	if err := os.MkdirAll(filepath.Dir(about), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ900\ntitle: About\nslug: about\nstatus: published\npublished_at: 2026-01-10T00:00:00Z\n---\n\nAbout.\n"
	if err := os.WriteFile(about, []byte(body), 0o644); err != nil {
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

	f.run(t, f.out, nil)

	for _, tc := range []struct {
		file       string
		prev, next string // links the page must and must not carry
		absent     []string
	}{
		{"posts/post-02/index.html", "/posts/post-01/", "", []string{`rel="next"`, "/about/"}},
		{"posts/post-01/index.html", "/posts/post-00/", "/posts/post-02/", []string{"/about/"}},
		{"posts/post-00/index.html", "", "/posts/post-01/", []string{`rel="prev"`, "/about/"}},
		{"about/index.html", "", "", []string{`rel="prev"`, `rel="next"`}},
	} {
		page := readFile(t, f.out, tc.file)
		if tc.prev != "" && !strings.Contains(page, `rel="prev" href="`+tc.prev+`"`) {
			t.Errorf("%s: does not link to the older post %s", tc.file, tc.prev)
		}
		if tc.next != "" && !strings.Contains(page, `rel="next" href="`+tc.next+`"`) {
			t.Errorf("%s: does not link to the newer post %s", tc.file, tc.next)
		}
		for _, no := range tc.absent {
			if strings.Contains(page, no) {
				t.Errorf("%s: must not contain %s", tc.file, no)
			}
		}
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

// A scheduled post used to be built as soon as it was saved, because a build
// chose what to publish by status alone: it was on the home page, in the feed
// and at its own address a month early.
func TestScheduledContentWaitsForItsTime(t *testing.T) {
	f := newFixture(t, 3)
	due := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	f.add(t, "content/posts/launch-day/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZ901
title: Launch Day
slug: launch-day
status: scheduled
published_at: 2026-07-01T09:00:00Z
tags: [Go, Soon]
---

Not yet.
`)

	stats, early := f.run(t, f.out, nil)
	if !stats.NextDue.Equal(due) {
		t.Errorf("Stats.NextDue = %v, want %v", stats.NextDue, due)
	}
	for _, file := range []string{"posts/launch-day/index.html", "tags/soon/index.html"} {
		if slices.Contains(early, file) {
			t.Errorf("%s was built before its time", file)
		}
	}
	for _, file := range []string{"index.html", "posts/index.html", "tags/index.html", "tags/go/index.html", "rss.xml", "sitemap.xml"} {
		page := readFile(t, f.out, file)
		for _, leak := range []string{"launch-day", "Launch Day", "tags/soon"} {
			if strings.Contains(page, leak) {
				t.Errorf("%s shows %q before its time", file, leak)
			}
		}
	}
	if got, err := f.builder(t, nil, nil).NextDue(t.Context()); err != nil || !got.Equal(due) {
		t.Errorf("NextDue = %v, %v; want %v", got, err, due)
	}

	onTime := filepath.Join(f.root, "on-time")
	_, files := f.run(t, onTime, func(o *build.Options) { o.Now = due })
	for _, file := range []string{"posts/launch-day/index.html", "tags/soon/index.html"} {
		if !slices.Contains(files, file) {
			t.Errorf("%s is missing once its time has come", file)
		}
	}
	for _, file := range []string{"index.html", "rss.xml", "sitemap.xml"} {
		if !strings.Contains(readFile(t, onTime, file), "launch-day") {
			t.Errorf("%s does not list the post once its time has come", file)
		}
	}
	if !strings.Contains(readFile(t, onTime, "tags/index.html"), "tags/soon") {
		t.Error("the tag index does not list the new term once its time has come")
	}
	if got, err := f.builder(t, nil, func(o *build.Options) { o.Now = due }).NextDue(t.Context()); err != nil || !got.IsZero() {
		t.Errorf("NextDue with nothing waiting = %v, %v; want zero", got, err)
	}

	// A preview with drafts shows what is coming, so nothing in it is waiting.
	_, preview := f.run(t, filepath.Join(f.root, "preview"), func(o *build.Options) { o.IncludeDrafts = true })
	if !slices.Contains(preview, "posts/launch-day/index.html") {
		t.Error("--drafts did not include the scheduled post")
	}
	if got, err := f.builder(t, nil, func(o *build.Options) { o.IncludeDrafts = true }).NextDue(t.Context()); err != nil || !got.IsZero() {
		t.Errorf("NextDue with drafts = %v, %v; want zero", got, err)
	}
}

// Term listings are grouped from the items the plan has already loaded
// rather than asked of the index one term at a time. What they list, and in
// what order, has to be exactly what the index would have said.
func TestTermListingsAgreeWithTheIndex(t *testing.T) {
	f := newFixture(t, 9)
	f.add(t, "content/posts/tagged/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZ901
title: Tagged
slug: tagged
status: published
published_at: 2026-02-01T00:00:00Z
tags: [Go, Notes, Go]
categories: [Tech]
---

Body.
`)
	plan, err := f.builder(t, nil, nil).Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var checked int
	for _, target := range plan.Targets {
		if target.Kind != render.KindTerm {
			continue
		}
		page, err := f.reader.Query(t.Context(), content.Query{
			PublicAt: &fixtureNow,
			TermsAny: map[string][]string{target.Type: {target.Term}},
			Limit:    content.MaxLimit,
		})
		if err != nil {
			t.Fatal(err)
		}
		var want []content.ID
		for _, s := range page.Items {
			want = append(want, s.ID)
		}
		lo := (target.Page - 1) * 3 // the fixture's page size
		want = want[lo:min(lo+3, len(want))]

		var got []content.ID
		for _, s := range target.Items {
			got = append(got, s.ID)
		}
		if !slices.Equal(got, want) || target.TotalItems != len(page.Items) {
			t.Errorf("%s/%s page %d lists %v of %d, the index says %v of %d",
				target.Type, target.Term, target.Page, got, target.TotalItems, want, len(page.Items))
		}
		checked++
	}
	if checked < 3 {
		t.Fatalf("only %d term pages were planned", checked)
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

// Titles were derived from whatever was at hand while rendering, which gave
// the home page a content type's name, left the error page with none, and
// printed internal taxonomy names as the author never wrote them.
func TestEveryPageKindIsTitledSensibly(t *testing.T) {
	f := newFixture(t, 4)
	_, _ = f.run(t, f.out, nil)

	for _, tc := range []struct {
		file  string
		title string
		why   string
	}{
		{"index.html", "Test", "the home page is the site, not one of its content types"},
		{"posts/index.html", "Posts · Test", "a listing is named after what it lists"},
		{"tags/index.html", "Tags · Test", "a taxonomy Kite named is presented, not printed raw"},
		{"tags/go/index.html", "Go · Test", "a term keeps the spelling its author used"},
		{"404.html", "Not found · Test", "an error page still has to say what it is"},
	} {
		got := documentTitle(t, readFile(t, f.out, tc.file))
		if got != tc.title {
			t.Errorf("%s: title = %q, want %q\n  (%s)", tc.file, got, tc.title, tc.why)
		}
	}
}

func TestPageWithoutATitleLeavesNoDanglingSeparator(t *testing.T) {
	f := newFixture(t, 2)
	_, _ = f.run(t, f.out, nil)

	title := documentTitle(t, readFile(t, f.out, "index.html"))
	if strings.HasPrefix(title, "·") || strings.Contains(title, " ·  ") {
		t.Errorf("title = %q, want no separator with nothing in front of it", title)
	}
}

func documentTitle(t *testing.T, page string) string {
	t.Helper()
	const open, close = "<title>", "</title>"
	i := strings.Index(page, open)
	j := strings.Index(page, close)
	if i < 0 || j < i {
		t.Fatal("page has no title element")
	}
	return strings.TrimSpace(page[i+len(open) : j])
}

// Under the extension url style every bundle in a section shares one output
// directory, so two items can own a file of the same name. Publishing one
// over the other would leave a page showing another page's picture.
func TestTwoBundlesCannotQuietlyPublishOneFile(t *testing.T) {
	plan := &build.Plan{Targets: []build.Target{
		{Kind: render.KindSingle, Path: "posts/one.html",
			Item: &content.Content{ID: "1", Locator: "content/posts/one"}},
		{Kind: render.KindSingle, Path: "posts/two.html",
			Item: &content.Content{ID: "2", Locator: "content/posts/two"}},
	}}

	media := fstest.MapFS{
		"content/posts/one/index.md":  {Data: []byte("x")},
		"content/posts/one/cover.png": {Data: []byte("one")},
		"content/posts/two/index.md":  {Data: []byte("x")},
		"content/posts/two/cover.png": {Data: []byte("two")},
	}

	_, err := build.MediaFiles(plan, media)
	if err == nil {
		t.Fatal("two bundles were allowed to publish one file")
	}
	for _, want := range []string{"content/posts/one", "content/posts/two", "cover.png"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not name %s: %v", want, err)
		}
	}
}

// The same two files under the directory style have addresses of their own.
func TestBundlesWithTheirOwnDirectoriesDoNotCollide(t *testing.T) {
	plan := &build.Plan{Targets: []build.Target{
		{Kind: render.KindSingle, Path: "posts/one/index.html",
			Item: &content.Content{ID: "1", Locator: "content/posts/one"}},
		{Kind: render.KindSingle, Path: "posts/two/index.html",
			Item: &content.Content{ID: "2", Locator: "content/posts/two"}},
	}}

	media := fstest.MapFS{
		"content/posts/one/cover.png": {Data: []byte("one")},
		"content/posts/two/cover.png": {Data: []byte("two")},
	}

	files, err := build.MediaFiles(plan, media)
	if err != nil {
		t.Fatalf("MediaFiles: %v", err)
	}
	want := map[string]string{
		"posts/one/cover.png": "content/posts/one/cover.png",
		"posts/two/cover.png": "content/posts/two/cover.png",
	}
	if !maps.Equal(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}
