package build_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
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
	baseURL string
	reader  *reader.Reader
	types   *content.Registry
	hooks   *hook.Bus
	resolve *kurl.Resolver
	engine  *theme.Engine
}

func newFixture(t *testing.T, posts int) *fixture {
	t.Helper()
	return newFixtureAt(t, posts, "https://example.com")
}

// newFixtureAt is a fixture site published at baseURL.
func newFixtureAt(t *testing.T, posts int, baseURL string) *fixture {
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
		BaseURL: baseURL, Style: kurl.StyleDirectory,
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
		baseURL: baseURL,
		reader:  reader.New(ix.DB()),
		types:   types,
		hooks:   bus,
		resolve: resolver,
		engine: theme.NewEngine(theme.Options{
			Sources: []theme.Source{{Name: "default", FS: th.Layouts}},
			Links:   resolver,
		}),
	}
}

// fixtureNow is the build clock unless a test sets its own.
var fixtureNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func (f *fixture) builder(t *testing.T, emitter *build.Emitter, mutate func(*build.Options)) *build.Builder {
	t.Helper()
	opts := build.Options{
		Site: render.SiteInfo{
			Title: "Test", BaseURL: f.baseURL, Language: "en",
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

// Hugo and Jekyll hold back a post dated in the future, and Kite reads a
// file that gives no status as published, with its date as the time it goes
// out. A published post dated later has to wait as a scheduled one does, or
// a site moved from either puts its future posts up at once.
func TestAPublishedPostDatedLaterWaitsForItsDate(t *testing.T) {
	f := newFixture(t, 3)
	f.add(t, "content/posts/from-hugo/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZ902
title: From Hugo
slug: from-hugo
date: 2026-08-01T09:00:00Z
tags: [Later]
---

Not yet.
`)
	f.add(t, "content/posts/launch-day/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZ901
title: Launch Day
slug: launch-day
status: scheduled
published_at: 2026-09-01T09:00:00Z
---

Later still.
`)
	due := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

	stats, early := f.run(t, f.out, nil)
	if !stats.NextDue.Equal(due) {
		t.Errorf("Stats.NextDue = %v, want the published post's %v", stats.NextDue, due)
	}
	for _, file := range []string{"posts/from-hugo/index.html", "tags/later/index.html"} {
		if slices.Contains(early, file) {
			t.Errorf("%s was built before its date", file)
		}
	}
	for _, file := range []string{"index.html", "posts/index.html", "tags/index.html", "rss.xml", "sitemap.xml"} {
		if page := readFile(t, f.out, file); strings.Contains(page, "from-hugo") || strings.Contains(page, "tags/later") {
			t.Errorf("%s shows the post before its date", file)
		}
	}

	onTime := filepath.Join(f.root, "on-time")
	stats, files := f.run(t, onTime, func(o *build.Options) { o.Now = due })
	for _, file := range []string{"posts/from-hugo/index.html", "tags/later/index.html"} {
		if !slices.Contains(files, file) {
			t.Errorf("%s is missing once its date has come", file)
		}
	}
	if !strings.Contains(readFile(t, onTime, "rss.xml"), "from-hugo") {
		t.Error("the feed does not list the post once its date has come")
	}
	if next := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC); !stats.NextDue.Equal(next) {
		t.Errorf("Stats.NextDue = %v, want the scheduled post's %v", stats.NextDue, next)
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
	f.add(t, "content/posts/lower/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZ902", "lower", "2026-02-02",
		"tags: [go, notes]\ncategories: [tech]\n"))
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

	// A taxonomy's listing names and counts the terms its term pages list.
	for _, taxonomy := range []string{"tags", "categories"} {
		counts, err := f.reader.CountTerms(t.Context(), taxonomy, content.Query{PublicAt: &fixtureNow})
		if err != nil {
			t.Fatal(err)
		}
		listed := make(map[string]int)
		for _, c := range counts {
			listed[c.Term] = c.Count
		}
		pages := make(map[string]int)
		for _, target := range plan.Targets {
			if target.Kind == render.KindTerm && target.Type == taxonomy && target.Page == 1 {
				pages[target.Term] = target.TotalItems
			}
		}
		if !maps.Equal(listed, pages) {
			t.Errorf("%s: the listing counts %v, the term pages list %v", taxonomy, listed, pages)
		}
	}
}

// post is the source of a published post, with more front matter in extra.
func post(id, slug, published, extra string) string {
	return fmt.Sprintf("---\nid: %s\ntitle: %s\nslug: %s\nstatus: published\npublished_at: %sT00:00:00Z\n%s---\n\nBody.\n",
		id, slug, slug, published, extra)
}

// Terms written differently that share an address are one term. Planned
// apart, Go and go were two pages written to one file, and the build kept
// whichever was written last: the other's posts were missing from the page,
// and which ones changed from one build to the next.
func TestTermsWrittenTwoWaysAreOnePage(t *testing.T) {
	f := newFixture(t, 2)
	f.add(t, "content/posts/lower/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZ901", "lower", "2026-02-01", "tags: [go]\n"))
	f.add(t, "content/posts/both/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZ902", "both", "2026-02-02", "tags: [GO, Go]\n"))
	roomy := func(o *build.Options) { o.PageSize = 10 }

	plan, err := f.builder(t, nil, roomy).Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	written := make(map[string]int)
	for _, target := range plan.Targets {
		if written[target.Path]++; written[target.Path] == 2 {
			t.Errorf("two targets are written to %s", target.Path)
		}
	}

	for i := range 3 {
		dir := filepath.Join(f.root, fmt.Sprintf("run%d", i))
		f.run(t, dir, roomy)

		term := readFile(t, dir, "tags/go/index.html")
		for _, post := range []string{"post-00", "post-01", "lower", "both"} {
			if !strings.Contains(term, `href="/posts/`+post+`/"`) {
				t.Errorf("run %d: tags/go does not list %s", i, post)
			}
		}
		if got := documentTitle(t, term); got != "Go · Test" {
			t.Errorf("run %d: tags/go is titled %q, want the way most posts write it", i, got)
		}
		listing := readFile(t, dir, "tags/index.html")
		if n := strings.Count(listing, `href="/tags/go/"`); n != 1 {
			t.Errorf("run %d: the tags listing links to tags/go %d times, want once", i, n)
		}
		if !strings.Contains(listing, `<a href="/tags/go/">Go</a><span class="count">4</span>`) {
			t.Errorf("run %d: the tags listing does not count Go on 4 posts:\n%s", i, excerptOf(listing, "tags/go"))
		}
	}

	// A post writing the term two ways names it once, as it first wrote it.
	single := readFile(t, filepath.Join(f.root, "run0"), "posts/both/index.html")
	if !strings.Contains(single, `<a href="/tags/go/">GO</a>`) || strings.Contains(single, `>Go</a>`) {
		t.Errorf("posts/both should show its term once, as GO:\n%s", excerptOf(single, "filed-in"))
	}
}

func TestATermIsCalledWhatMostOfItsPostsWrite(t *testing.T) {
	for _, tc := range []struct {
		name string
		tags []string // one post each
		want string
		link string
	}{
		{"the most used", []string{"go", "Go", "go"}, "go", "/tags/go/"},
		{"used alike, the first in byte order", []string{"go", "Go"}, "Go", "/tags/go/"},
		{"spaced", []string{"web-dev", "Web Dev", "web  dev"}, "Web Dev", "/tags/web-dev/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, 0)
			for i, tag := range tc.tags {
				id := fmt.Sprintf("01J8KQ2P3R4S5T6V7W8X9YZ9%02d", i)
				f.add(t, fmt.Sprintf("content/posts/p%d/index.md", i),
					post(id, fmt.Sprintf("p%d", i), fmt.Sprintf("2026-02-%02d", i+1), "tags: ["+tag+"]\n"))
			}
			plan, err := f.builder(t, nil, nil).Plan(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			var terms []string
			for _, target := range plan.Targets {
				if target.Kind != render.KindTerm {
					continue
				}
				terms = append(terms, target.Term)
				if target.Title != tc.want || target.URL != tc.link || target.TotalItems != len(tc.tags) {
					t.Errorf("term page %q at %s lists %d, want %q at %s listing %d",
						target.Title, target.URL, target.TotalItems, tc.want, tc.link, len(tc.tags))
				}
			}
			if len(terms) != 1 {
				t.Errorf("term pages %v, want one", terms)
			}

			counts, err := f.reader.CountTerms(t.Context(), "tags", content.Query{})
			if err != nil {
				t.Fatal(err)
			}
			if len(counts) != 1 || counts[0].Term != tc.want || counts[0].Count != len(tc.tags) {
				t.Errorf("the index counts %+v, want %s on %d", counts, tc.want, len(tc.tags))
			}
		})
	}
}

// A term of nothing but dashes, slashes or spaces has nothing to write in
// its address, which would then be the taxonomy's own listing.
func TestATermWithAnEmptySlugHasNoPage(t *testing.T) {
	f := newFixture(t, 1)
	f.add(t, "content/posts/dash/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZ901", "dash", "2026-02-01", `tags: ["-", Go]`+"\n"))
	_, files := f.run(t, f.out, nil)

	if !slices.Contains(files, "tags/index.html") || !slices.Contains(files, "tags/go/index.html") {
		t.Fatalf("the tags pages are missing\ngot: %v", files)
	}
	if listing := readFile(t, f.out, "tags/index.html"); !strings.Contains(listing, `class="terms"`) ||
		strings.Contains(listing, ">-</a>") {
		t.Errorf("tags/index.html should be the listing, without the dash:\n%s", excerptOf(listing, `class="terms"`))
	}
	if single := readFile(t, f.out, "posts/dash/index.html"); strings.Contains(single, ">-</a>") {
		t.Errorf("posts/dash links its dash to the listing:\n%s", excerptOf(single, "filed-in"))
	}
}

// A failed build must leave the previous site in place rather than a mixture.
func TestFailedBuildLeavesPreviousOutputIntact(t *testing.T) {
	f := newFixture(t, 2)
	f.run(t, f.out, nil)
	before := readFile(t, f.out, "index.html")

	broken := theme.NewEngine(theme.Options{Sources: []theme.Source{
		{Name: "broken", FS: os.DirFS(t.TempDir())},
	}, Links: f.resolve})
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

// A GitHub Pages project site without a domain of its own is published at
// /<repository>/. The files go where they would for a site at the root, since
// the host maps the path onto the published directory, and every link has to
// carry the path.
func TestASiteUnderAPathLinksWithinIt(t *testing.T) {
	f := newFixtureAt(t, 5, "https://example.github.io/blog/")
	_, files := f.run(t, f.out, nil)

	for _, want := range []string{"index.html", "page/2/index.html", "posts/post-00/index.html", "tags/go/index.html", "404.html"} {
		if !slices.Contains(files, want) {
			t.Errorf("%s was not written; the build wrote %v", want, files)
		}
	}

	checked := 0
	for _, name := range files {
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		for _, m := range rootRelativeLink.FindAllStringSubmatch(readFile(t, f.out, name), -1) {
			checked++
			if link := m[1]; link != "/blog" && !strings.HasPrefix(link, "/blog/") {
				t.Errorf("%s links to %s, outside the site at /blog/", name, link)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no link was checked")
	}

	for _, name := range []string{"sitemap.xml", "rss.xml"} {
		found := absoluteLink.FindAllStringSubmatch(readFile(t, f.out, name), -1)
		if len(found) == 0 {
			t.Errorf("%s holds no address", name)
		}
		for _, m := range found {
			if !strings.HasPrefix(m[1], "https://example.github.io/blog/") {
				t.Errorf("%s gives %s, outside the site", name, m[1])
			}
		}
	}
}

// rootRelativeLink matches a link that starts at the root of its host.
var rootRelativeLink = regexp.MustCompile(`(?:href|src)="(/(?:[^/"][^"]*)?)"`)

// absoluteLink matches the addresses a sitemap or a feed gives.
var absoluteLink = regexp.MustCompile(`<(?:loc|link)>([^<]*)</(?:loc|link)>`)

// A server asks for what the completion hooks write without drawing any page,
// and has to be given the bytes a build writes, observers included.
func TestExtrasAreWhatABuildWrites(t *testing.T) {
	f := newFixture(t, 5)
	bus := hook.NewBus()
	bus.Register(retitling{hook.Base{HookName: "retitling", HookPhase: hook.PhaseBuild}}, hook.DefaultPriority)
	builtin.Register(bus, builtin.DefaultOptions())
	withBus := func(o *build.Options) { o.Hooks = bus }

	_, files := f.run(t, f.out, withBus)
	if !strings.Contains(readFile(t, f.out, "rss.xml"), "Observed:") {
		t.Fatal("the observer did not reach the feed")
	}

	b := f.builder(t, nil, withBus)
	plan, err := b.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	extras, err := b.Extras(t.Context(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := slices.Sorted(maps.Keys(extras)); !slices.Equal(got, []string{"rss.xml", "sitemap.xml"}) {
		t.Fatalf("extras = %v", got)
	}
	for name, data := range extras {
		if !slices.Contains(files, name) {
			t.Errorf("%s is not something the build wrote", name)
		} else if want := readFile(t, f.out, name); string(data) != want {
			t.Errorf("%s differs from the built one:\n%s\nwant:\n%s", name, data, want)
		}
	}
}

// A hook that writes outside the output is refused whichever runtime asked.
func TestExtrasRefuseWhatABuildRefuses(t *testing.T) {
	f := newFixture(t, 1)
	bus := hook.NewBus()
	bus.Register(escaping{hook.Base{HookName: "escaping", HookPhase: hook.PhaseBuild}}, hook.DefaultPriority)
	b := f.builder(t, nil, func(o *build.Options) { o.Hooks = bus })

	plan, err := b.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Extras(t.Context(), plan); err == nil || !strings.Contains(err.Error(), "outside the output") {
		t.Errorf("err = %v, want the refusal a build gives", err)
	}
}

// An editor shows a count before its item is saved, and it has to be the one
// the page will show, so the markdown hooks run first, as they do for a page.
func TestAWordCountGoesThroughTheMarkdownHooks(t *testing.T) {
	f := newFixture(t, 1)
	bus := hook.NewBus()
	bus.Register(signing{hook.Base{HookName: "signing", HookPhase: hook.PhaseBuild}}, hook.DefaultPriority)
	b := f.builder(t, nil, func(o *build.Options) { o.Hooks = bus })

	item := &content.Content{
		Kind: "post",
		Body: content.Body{Format: content.FormatMarkdown, Raw: "Two words\n"},
	}
	words, err := b.WordCount(t.Context(), item)
	if err != nil {
		t.Fatal(err)
	}
	if words != 5 {
		t.Errorf("WordCount = %d, want 5 with the three words the hook signs", words)
	}
}

// A listing says of an item what the item's own page says. The index keeps a
// summary of every item, but knows nothing of hooks, so when one rewrites
// markdown the listing asks the page instead.
func TestAListingSaysWhatThePageSaysThroughTheHooks(t *testing.T) {
	f := newFixture(t, 2)
	described := filepath.Join(f.root, "content", "posts", "post-01", "index.md")
	data, err := os.ReadFile(described)
	if err != nil {
		t.Fatal(err)
	}
	withDescription := strings.Replace(string(data), "tags: [Go]\n", "tags: [Go]\ndescription: Written to stand *alone*.\n", 1)
	if err := os.WriteFile(described, []byte(withDescription), 0o644); err != nil {
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

	bus := hook.NewBus()
	bus.Register(signing{hook.Base{HookName: "signing", HookPhase: hook.PhaseBuild}}, hook.DefaultPriority)
	f.run(t, f.out, func(o *build.Options) {
		o.Hooks = bus
		o.Site.ThemeSettings = map[string]any{"show_summary": true}
	})
	home, err := os.ReadFile(filepath.Join(f.out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "Body of post 00. Signed by Kite.") {
		t.Error("the listing does not carry what the hook added to the post")
	}
	if !strings.Contains(string(home), "Written to stand alone.") || strings.Contains(string(home), "Body of post 01.") {
		t.Error("a post with a description is not listed by it")
	}
}

// A page that reaches a template through a listing, or as the neighbor of a
// single page, carries what its own page does: what the author set on it,
// such as a cover, how long it takes to read and the pictures it shows, hooks
// and all. The pictures are as the body writes them, so a theme resolving one
// on a site under a path does not add the path twice.
func TestAListedPageCarriesItsParamsLengthAndPictures(t *testing.T) {
	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	const show = `{{ .Title }}|{{ .Params.cover }}|{{ .Params.style.tone }}|{{ .WordCount }}|{{ .ReadingTime.Minutes }}|{{ .Images }};`
	layouts := fstest.MapFS{
		"home.html": {Data: []byte(`{{ define "main" }}{{ range .Pages }}` + show + `{{ end }}{{ end }}`)},
		"single.html": {Data: []byte(`{{ define "main" }}{{ with .Page }}` + show + `{{ .Content }}{{ end }}` +
			`{{ with .Page.Next }}next:` + show + `{{ end }}{{ end }}`)},
	}
	// 1200 characters at 400 a minute and 220 words at 220 a minute.
	body := "![first](/uploads/first.jpg)\n\n" + strings.Repeat("桂花开了 ", 300) + strings.Repeat("word ", 220) +
		"\n\n![second](second.png)\n"

	for _, tc := range []struct {
		name   string
		signed bool
		want   string
	}{
		{"as indexed", false, "Covered|cover.jpg|warm|1420|4|[/uploads/first.jpg second.png];"},
		{"through a markdown hook", true, "Covered|cover.jpg|warm|1423|5|[/uploads/first.jpg second.png];"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixtureAt(t, 2, "https://example.com/blog/")
			f.engine = theme.NewEngine(theme.Options{
				Sources: []theme.Source{{Name: "site", FS: layouts}, {Name: "default", FS: th.Layouts}},
				Links:   f.resolve,
			})
			f.add(t, "content/posts/covered/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ950\ntitle: Covered\nslug: covered\n"+
				"status: published\npublished_at: 2026-01-20T00:00:00Z\ncover: cover.jpg\nstyle: {tone: warm}\n---\n\n"+body)
			f.run(t, f.out, func(o *build.Options) {
				o.Markdown = markdown.New(markdown.Options{BasePath: "/blog/"})
				if tc.signed {
					o.Hooks = hook.NewBus()
					o.Hooks.Register(signing{hook.Base{HookName: "signing", HookPhase: hook.PhaseBuild}}, hook.DefaultPriority)
				}
			})

			page := readFile(t, f.out, "posts/covered/index.html")
			if !strings.Contains(page, tc.want) {
				t.Fatalf("the post's own page does not say %q: %s", tc.want, excerptOf(page, "Covered|"))
			}
			if !strings.Contains(page, `src="/blog/uploads/first.jpg"`) {
				t.Errorf("the post does not show its picture under the site's path: %s", excerptOf(page, "<img"))
			}
			if home := readFile(t, f.out, "index.html"); !strings.Contains(home, tc.want) {
				t.Errorf("the home page lists %q, want %q", excerptOf(home, "Covered|"), tc.want)
			}
			if older := readFile(t, f.out, "posts/post-01/index.html"); !strings.Contains(older, "next:"+tc.want) {
				t.Errorf("the older post's neighbor is %q, want %q", excerptOf(older, "next:"), tc.want)
			}
		})
	}
}

type signing struct{ hook.Base }

func (signing) TransformMarkdown(_ context.Context, doc *hook.MarkdownDoc) error {
	doc.Source += "\nSigned by Kite.\n"
	return nil
}

type retitling struct{ hook.Base }

func (retitling) PageRendered(_ context.Context, p *hook.PageInfo) error {
	p.Title = "Observed: " + p.Title
	return nil
}

type escaping struct{ hook.Base }

func (escaping) BuildComplete(_ context.Context, b *hook.BuildInfo) error {
	return b.Emit("../outside.xml", []byte("x"))
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

// A bundle that keeps its pictures in a folder, as many Hugo sites do, has
// them published at the same place under the page, or ![](images/01.png) is
// a broken picture. A folder that is another item's bundle publishes with it,
// not twice.
func TestABundlesFoldersArePublishedWithIt(t *testing.T) {
	plan := &build.Plan{Targets: []build.Target{
		{Kind: render.KindSingle, Path: "posts/trip/index.html",
			Item: &content.Content{ID: "1", Locator: "content/posts/trip"}},
		{Kind: render.KindSingle, Path: "posts/day/index.html",
			Item: &content.Content{ID: "2", Locator: "content/posts/trip/day"}},
		{Kind: render.KindSingle, Path: "posts/flat/index.html",
			Item: &content.Content{ID: "3", Locator: "content/posts/flat.md"}},
	}}
	media := fstest.MapFS{
		"content/posts/trip/index.md":          {Data: []byte("x")},
		"content/posts/trip/cover.jpg":         {Data: []byte("cover")},
		"content/posts/trip/images/01.png":     {Data: []byte("one")},
		"content/posts/trip/images/raw/02.png": {Data: []byte("two")},
		"content/posts/trip/images/notes.md":   {Data: []byte("a source, not published")},
		"content/posts/trip/.hidden/x.png":     {Data: []byte("hidden")},
		"content/posts/trip/day/index.md":      {Data: []byte("x")},
		"content/posts/trip/day/sea.jpg":       {Data: []byte("sea")},
		"content/posts/flat.md":                {Data: []byte("x")},
	}

	files, err := build.MediaFiles(plan, media)
	if err != nil {
		t.Fatalf("MediaFiles: %v", err)
	}
	want := map[string]string{
		"posts/trip/cover.jpg":         "content/posts/trip/cover.jpg",
		"posts/trip/images/01.png":     "content/posts/trip/images/01.png",
		"posts/trip/images/raw/02.png": "content/posts/trip/images/raw/02.png",
		"posts/day/sea.jpg":            "content/posts/trip/day/sea.jpg",
	}
	if !maps.Equal(files, want) {
		t.Errorf("files = %v\nwant %v", files, want)
	}
}

// Under the extension style two bundles' folders land in one directory too,
// and a file both would publish is refused by its whole path.
func TestTwoBundlesCannotQuietlyPublishOneFileInAFolder(t *testing.T) {
	plan := &build.Plan{Targets: []build.Target{
		{Kind: render.KindSingle, Path: "posts/one.html",
			Item: &content.Content{ID: "1", Locator: "content/posts/one"}},
		{Kind: render.KindSingle, Path: "posts/two.html",
			Item: &content.Content{ID: "2", Locator: "content/posts/two"}},
	}}
	media := fstest.MapFS{
		"content/posts/one/images/a.png": {Data: []byte("one")},
		"content/posts/two/images/a.png": {Data: []byte("two")},
		"content/posts/two/images/b.png": {Data: []byte("two")},
	}
	_, err := build.MediaFiles(plan, media)
	if err == nil || !strings.Contains(err.Error(), "images/a.png") {
		t.Errorf("err = %v, want a refusal naming images/a.png", err)
	}
}

// A page is drawn with the layout its front matter names, as Hugo writes it.
// A page that names none, or names something that could not be a template's
// file name, keeps its type's own template.
func TestAPageIsDrawnWithTheLayoutItNames(t *testing.T) {
	f := newFixture(t, 1)

	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	site := fstest.MapFS{
		"page/links.html": {Data: []byte(`{{ define "main" }}<ul class="drawn-as-links">{{ .Page.Title }}</ul>{{ end }}`)},
	}
	f.engine = theme.NewEngine(theme.Options{
		Sources: []theme.Source{{Name: "site", FS: site}, {Name: "default", FS: th.Layouts}},
		Links:   f.resolve,
	})

	page := func(id, slug, layout string) string {
		return fmt.Sprintf("---\nid: %s\ntitle: %s\nslug: %s\nstatus: published\n"+
			"published_at: 2026-01-10T00:00:00Z\n%s---\n\nBody.\n", id, slug, slug, layout)
	}
	f.add(t, "content/pages/friends.md", page("01J8KQ2P3R4S5T6V7W8X9YZ901", "friends", "layout: links\n"))
	f.add(t, "content/pages/about.md", page("01J8KQ2P3R4S5T6V7W8X9YZ902", "about", ""))
	f.add(t, "content/pages/escape.md", page("01J8KQ2P3R4S5T6V7W8X9YZ903", "escape", "layout: ../page/links\n"))

	f.run(t, f.out, nil)

	if !strings.Contains(readFile(t, f.out, "friends/index.html"), `class="drawn-as-links"`) {
		t.Error("a page naming layout links was not drawn with page/links.html")
	}
	for _, file := range []string{"about/index.html", "escape/index.html"} {
		if strings.Contains(readFile(t, f.out, file), "drawn-as-links") {
			t.Errorf("%s was drawn with a layout it did not validly name", file)
		}
	}
}

// An address that changed has to keep working, which is what aliases say, and
// a static host can only redirect with a page: one that sends a browser on and
// tells a search engine which address to keep, at every alias. It belongs to
// nobody's listing, sitemap or feed.
func TestAnAliasSendsAReaderOnToWhereTheItemIsNow(t *testing.T) {
	f := newFixtureAt(t, 1, "https://example.com/blog/")
	f.add(t, "content/posts/trip/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZTRP
title: A <trip>
status: published
published_at: 2026-02-01T00:00:00Z
aliases: [/travel/trip/, old-trip, /2019/05/trip.html, "/旅行/", /blog-posts/trip, /posts/trip/, /posts/trip.html]
---
Body.
`)
	_, files := f.run(t, f.out, nil)

	for _, want := range []string{
		"travel/trip/index.html",
		"posts/old-trip/index.html",
		"2019/05/trip.html",
		"旅行/index.html",
		"blog-posts/trip/index.html",
		"posts/trip.html",
	} {
		if !slices.Contains(files, want) {
			t.Errorf("no page at the alias %s\ngot: %v", want, files)
		}
	}

	page := readFile(t, f.out, "travel/trip/index.html")
	for _, want := range []string{
		`<meta http-equiv="refresh" content="0; url=/blog/posts/trip/">`,
		`<link rel="canonical" href="https://example.com/blog/posts/trip/">`,
		`<meta name="robots" content="noindex">`,
		`<a href="/blog/posts/trip/">A &lt;trip&gt;</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the alias page lacks %s:\n%s", want, page)
		}
	}
	for _, listing := range []string{"sitemap.xml", "rss.xml", "index.html"} {
		if strings.Contains(readFile(t, f.out, listing), "travel") {
			t.Errorf("%s lists an alias", listing)
		}
	}
}

// An alias at another page's address would leave one of the two unreachable,
// and nothing would say which, so the build says so instead.
func TestAnAliasCannotTakeAnotherPagesAddress(t *testing.T) {
	for _, alias := range []string{"/posts/post-00/", "/tags/", "/"} {
		t.Run(alias, func(t *testing.T) {
			f := newFixture(t, 1)
			f.add(t, "content/posts/trip/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZTRP
title: Trip
status: published
published_at: 2026-02-01T00:00:00Z
aliases: ["`+alias+`"]
---
Body.
`)
			emitter, err := build.NewEmitter(f.out)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.builder(t, emitter, nil).Run(t.Context())
			if err == nil || !strings.Contains(err.Error(), "content/posts/trip") {
				t.Errorf("err = %v, want a refusal naming the item", err)
			}
		})
	}
}

// Two items may not claim one old address, and an alias must be a path.
func TestAnAliasIsOneItemsPathWithinTheSite(t *testing.T) {
	for name, aliases := range map[string][2]string{
		"claimed twice": {"/old/", "/old/"},
		"a full URL":    {"https://elsewhere.example/x/", "/fine/"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, 0)
			for i, alias := range aliases {
				f.add(t, fmt.Sprintf("content/posts/p%d/index.md", i), fmt.Sprintf(`---
id: 01J8KQ2P3R4S5T6V7W8X9YZP%02d
title: P%d
status: published
published_at: 2026-02-0%dT00:00:00Z
aliases: [%q]
---
Body.
`, i, i, i+1, alias))
			}
			emitter, err := build.NewEmitter(f.out)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.builder(t, emitter, nil).Run(t.Context()); err == nil {
				t.Error("the build went ahead")
			}
		})
	}
}

// Feed readers do not follow a page that redirects, so a site that moved from
// Hugo keeps its subscribers by writing the same feed where they subscribed.
func TestTheFeedIsAlsoWrittenToItsAliases(t *testing.T) {
	f := newFixture(t, 2)
	f.hooks = hook.NewBus()
	opts := builtin.DefaultOptions()
	opts.FeedAliases = []string{"index.xml", "posts/index.xml"}
	builtin.Register(f.hooks, opts)

	_, files := f.run(t, f.out, nil)
	feed := readFile(t, f.out, "rss.xml")
	for _, alias := range opts.FeedAliases {
		if !slices.Contains(files, alias) {
			t.Fatalf("no feed at %s\ngot: %v", alias, files)
		}
		if got := readFile(t, f.out, alias); got != feed {
			t.Errorf("%s differs from rss.xml", alias)
		}
	}
	if strings.Contains(readFile(t, f.out, "sitemap.xml"), "index.xml") {
		t.Error("the sitemap lists a feed")
	}
}

// A date in front matter reaches a template as a time and a count as an int,
// on the item's own page and in a list alike, in YAML and in TOML, so a theme
// formats a date and compares a count without parsing text.
func TestFrontMatterKeepsItsTypesInTemplates(t *testing.T) {
	f := newFixture(t, 0)
	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	const types = `{{ printf "%T %T %T %T" .Params.day (index .Params.events 0).at .Params.count .Params.rating }}`
	site := fstest.MapFS{
		"post/single.html": {Data: []byte(`{{ define "main" }}<p id="types">{{ with .Page }}` + types + `{{ end }}</p>{{ end }}`)},
		"post/list.html":   {Data: []byte(`{{ define "main" }}{{ range .Pages }}<p class="types">` + types + `</p>{{ end }}{{ end }}`)},
	}
	f.engine = theme.NewEngine(theme.Options{
		Sources: []theme.Source{{Name: "site", FS: site}, {Name: "default", FS: th.Layouts}},
		Links:   f.resolve,
	})
	f.add(t, "content/posts/yaml/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZYML
title: In YAML
status: published
published_at: 2026-02-01T00:00:00Z
day: 2026-03-04
events:
  - at: 2026-03-05 09:30:00
count: 3
rating: 4.0
---
Body.
`)
	f.add(t, "content/posts/toml/index.md", `+++
id = "01J8KQ2P3R4S5T6V7W8X9YZTML"
title = "In TOML"
status = "published"
published_at = 2026-02-02T00:00:00Z
day = 2026-03-04
events = [{at = 2026-03-05T09:30:00}]
count = 3
rating = 4.0
+++
Body.
`)
	f.run(t, f.out, nil)

	printed := regexp.MustCompile(`types">([^<]*)<`)
	for page, n := range map[string]int{"posts/yaml/index.html": 1, "posts/toml/index.html": 1, "posts/index.html": 2} {
		found := printed.FindAllStringSubmatch(readFile(t, f.out, page), -1)
		if len(found) != n {
			t.Errorf("%s: %d items printed their types, want %d", page, len(found), n)
		}
		for _, m := range found {
			if m[1] != "time.Time time.Time int float64" {
				t.Errorf("%s: types are %q", page, m[1])
			}
		}
	}
}

// A page can live under a path, and one whose address another page already
// has is refused rather than published over it or under it.
func TestAPageSlugMayBeAPathButNotAnotherPagesAddress(t *testing.T) {
	page := func(id, slug string) string {
		return fmt.Sprintf("---\nid: %s\ntitle: %s\nslug: %s\nstatus: published\n"+
			"published_at: 2026-01-10T00:00:00Z\n---\n\nBody.\n", id, slug, slug)
	}
	f := newFixture(t, 1)
	f.add(t, "content/pages/tideline.md", page("01J8KQ2P3R4S5T6V7W8X9YZ901", "projects/tideline"))
	_, files := f.run(t, f.out, nil)
	if !slices.Contains(files, "projects/tideline/index.html") {
		t.Errorf("no page under its path\ngot: %v", files)
	}

	for _, slug := range []string{"posts", "tags/go", "page/2", "posts/post-00"} {
		t.Run(slug, func(t *testing.T) {
			f := newFixture(t, 4)
			f.add(t, "content/pages/clash.md", page("01J8KQ2P3R4S5T6V7W8X9YZ902", slug))
			emitter, err := build.NewEmitter(f.out)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.builder(t, emitter, nil).Run(t.Context())
			if err == nil || !strings.Contains(err.Error(), "content/pages/clash.md") {
				t.Errorf("err = %v, want a refusal naming the page", err)
			}
		})
	}
}

// A body calls the shortcodes the site and its theme define, the site's
// before the theme's, and a shortcode's template is handed the call, the page
// and the site, and can call the theme's partials as a page can.
func TestABodyCallsTheShortcodesOfTheSiteAndItsTheme(t *testing.T) {
	f := newFixture(t, 1)
	fallback, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	th := fstest.MapFS{
		"single.html":              {Data: []byte(`<main>{{ .Page.Content }}</main>`)},
		"_partials/frame.html":     {Data: []byte(`<figure>{{ . }}</figure>`)},
		"_shortcodes/pic.html":     {Data: []byte(`theme pic`)},
		"_shortcodes/caption.html": {Data: []byte(`{{ partial "frame.html" (.Get "text") }}`)},
	}
	site := fstest.MapFS{
		"_shortcodes/pic.html": {Data: []byte(`<img src="{{ .Get 0 }}" alt="{{ .Page.Title }} on {{ .Site.Title }}" data-n="{{ .Ordinal }}"` +
			`{{ with .Parent }} data-in="{{ .Name }}"{{ end }}>`)},
		"_shortcodes/gallery.html": {Data: []byte(`<div class="gallery">{{ .Inner }}</div>`)},
	}
	f.engine = theme.NewEngine(theme.Options{
		Sources: []theme.Source{{Name: "site", FS: site}, {Name: "theme", FS: th}, {Name: "default", FS: fallback.Layouts}},
		Links:   f.resolve,
	})
	f.add(t, "content/posts/trip/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZTRP", "trip", "2026-01-20T00:00:00Z", "")+
		"{{< gallery >}}\n{{< pic \"a.jpg\" >}}\n{{< pic \"b.jpg\" >}}\n{{< /gallery >}}\n\n{{< caption text=\"By the river\" >}}\n")

	f.run(t, f.out, nil)
	page := readFile(t, f.out, "posts/trip/index.html")
	for _, want := range []string{
		`<div class="gallery"><img src="a.jpg" alt="trip on Test" data-n="0" data-in="gallery">` + "\n" +
			`<img src="b.jpg" alt="trip on Test" data-n="1" data-in="gallery">` + "\n</div>",
		`<figure>By the river</figure>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not hold %q:\n%s", want, page)
		}
	}
}

// A listing says of an item what its page says, and a shortcode that keeps
// what it encloses off the page keeps it out of both. The index cannot know
// what a template shows, so the listing asks the page.
func TestAListingCountsOnlyTheWordsAShortcodeShows(t *testing.T) {
	f := newFixture(t, 1)
	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	const show = `{{ .Title }}|{{ .WordCount }}|{{ .Excerpt }};`
	layouts := fstest.MapFS{
		"home.html":                {Data: []byte(`{{ define "main" }}{{ range .Pages }}` + show + `{{ end }}{{ end }}`)},
		"single.html":              {Data: []byte(`{{ define "main" }}{{ with .Page }}` + show + `{{ end }}{{ end }}`)},
		"_shortcodes/private.html": {Data: []byte(``)},
		"_shortcodes/mark.html":    {Data: []byte(`<mark>{{ .Inner }}</mark>`)},
	}
	f.engine = theme.NewEngine(theme.Options{
		Sources: []theme.Source{{Name: "site", FS: layouts}, {Name: "default", FS: th.Layouts}},
		Links:   f.resolve,
	})
	f.add(t, "content/posts/kept/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZKPT", "kept", "2026-01-20T00:00:00Z", "")+
		"Four {{< mark >}}words are{{< /mark >}} shown {{< private >}}and five are not shown{{< /private >}}.\n")

	f.run(t, f.out, nil)
	const want = "kept|5|Body. Four words are shown .;"
	if page := readFile(t, f.out, "posts/kept/index.html"); !strings.Contains(page, want) {
		t.Errorf("the page says %q, want %q", excerptOf(page, "kept|"), want)
	}
	if home := readFile(t, f.out, "index.html"); !strings.Contains(home, want) {
		t.Errorf("the home page lists %q, want %q", excerptOf(home, "kept|"), want)
	}
}

// A shortcode nobody defines stops the build at its file and line, rather than
// publishing the tag as text. A server shows the problem on the item's page
// and lists the item as the index knows it, so the rest of the site can
// still be looked at.
func TestAnUndefinedShortcodeStopsTheBuildAtItsLine(t *testing.T) {
	f := newFixture(t, 1)
	th, err := theme.Load(themes.Default())
	if err != nil {
		t.Fatal(err)
	}
	f.engine = theme.NewEngine(theme.Options{
		Sources: []theme.Source{
			{Name: "site", FS: fstest.MapFS{"_shortcodes/note.html": {Data: []byte(`{{ .Inner }}`)}}},
			{Name: "default", FS: th.Layouts},
		},
		Links: f.resolve,
	})
	f.add(t, "content/posts/trip/index.md", post("01J8KQ2P3R4S5T6V7W8X9YZTRP", "trip", "2026-01-20T00:00:00Z", "")+
		"First paragraph.\n\n{{< gallery >}}\n")

	emitter, err := build.NewEmitter(f.out)
	if err != nil {
		t.Fatal(err)
	}
	b := f.builder(t, emitter, func(o *build.Options) {
		o.Media = os.DirFS(f.root)
		o.Site.ThemeSettings = map[string]any{"show_summary": true}
	})
	_, err = b.Run(t.Context())
	const want = "content/posts/trip/index.md:12: shortcode gallery: not defined: add layouts/_shortcodes/gallery.html to the site or the theme"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want one containing %q", err, want)
	}
	if !errors.Is(err, content.ErrInvalid) {
		t.Errorf("%v is not reported as invalid content", err)
	}

	plan, err := b.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range plan.Targets {
		if target.Kind != render.KindHome {
			continue
		}
		home, _, err := b.Render(t.Context(), target, nil)
		if err != nil {
			t.Fatalf("the home page failed with the post: %v", err)
		}
		if !strings.Contains(string(home), "First paragraph.") {
			t.Errorf("the home page does not list the post as the index knows it")
		}
	}
	extras, err := b.Extras(t.Context(), plan)
	if err != nil {
		t.Fatalf("the feed and the sitemap failed with the post: %v", err)
	}
	if !strings.Contains(string(extras["rss.xml"]), "First paragraph.") {
		t.Errorf("the feed does not hold the post as the index knows it:\n%s", extras["rss.xml"])
	}
}
