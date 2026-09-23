package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/content"
)

func newTestProject(t *testing.T) (root string, types *content.Registry, w *Writer) {
	t.Helper()
	root = t.TempDir()
	types = content.DefaultRegistry()
	w = NewWriter(root, types)
	return root, types, w
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPutCreatesBundle(t *testing.T) {
	root, _, w := newTestProject(t)

	item := &content.Content{
		Kind:       "post",
		Title:      "Hello World",
		Status:     content.StatusDraft,
		Body:       content.Body{Format: content.FormatMarkdown, Raw: "\nbody text\n"},
		Taxonomies: map[string][]string{"tags": {"Go"}},
	}
	res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	want := "content/posts/hello-world/index.md"
	if !slices.Equal(res.Written, []string{want}) {
		t.Errorf("Written = %v, want [%s]", res.Written, want)
	}
	if item.ID == "" || !content.ValidID(string(item.ID)) {
		t.Errorf("id not minted: %q", item.ID)
	}
	if item.Locator != "content/posts/hello-world" {
		t.Errorf("Locator = %q", item.Locator)
	}

	got := readFile(t, root, want)
	for _, must := range []string{
		"id: " + string(item.ID),
		"title: Hello World",
		"slug: hello-world",
		"status: draft",
		"tags:\n  - Go",
		"body text",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("file missing %q:\n%s", must, got)
		}
	}
}

func TestPutIsIdempotentOnDisk(t *testing.T) {
	root, _, w := newTestProject(t)
	item := &content.Content{
		Kind:   "post",
		Title:  "Stable",
		Status: content.StatusPublished,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: "\nbody\n"},
	}
	ctx := t.Context()
	if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}}); err != nil {
		t.Fatal(err)
	}
	src := SourcePath(content.DefaultRegistry().Get("post"), item.Locator)
	first := readFile(t, root, src)

	// Re-saving an unchanged item must only move updated_at; nothing else may
	// shift, or every save would produce noise in git.
	if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{
		content.PutContent{Content: item, IfRevision: item.Revision},
	}}); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, root, src)

	changed := diffLines(first, second)
	for _, line := range changed {
		if !strings.Contains(line, "updated_at") {
			t.Errorf("unexpected change on re-save: %q", line)
		}
	}
}

func TestEditingTitleRewritesOnlyTitleLine(t *testing.T) {
	root, types, w := newTestProject(t)

	// A hand authored file with comments, flow lists and a custom key: exactly
	// the shape a user's repository has.
	const src = `---
id: 01J8KQ2P3R4S5T6V7W8X9YZABC
title: Original Title
slug: original
status: published
created_at: 2026-01-01T00:00:00Z
updated_at: 2026-01-01T00:00:00Z
tags: [Go, CMS]

# grouping for the docs site
category:
  - Tech
custom_key: keep me
---

Body stays put.
`
	writeFile(t, root, "content/posts/original/index.md", src)

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(scan.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(scan.Entries))
	}
	item := scan.Entries[0].Item
	item.Title = "Renamed Title"

	if _, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.PutContent{Content: item, IfRevision: item.Revision},
	}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got := readFile(t, root, "content/posts/original/index.md")
	changed := diffLines(src, got)

	for _, line := range changed {
		if !strings.Contains(line, "title:") && !strings.Contains(line, "updated_at:") {
			t.Errorf("unexpected line changed: %q", line)
		}
	}
	for _, must := range []string{
		"tags: [Go, CMS]",
		"# grouping for the docs site",
		"custom_key: keep me",
		"Body stays put.",
		"title: Renamed Title",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("lost %q:\n%s", must, got)
		}
	}
}

func TestRevisionConflict(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/a/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: a\n---\nbody\n")

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatal(err)
	}
	item := scan.Entries[0].Item
	item.Title = "B"

	_, err = w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.PutContent{Content: item, IfRevision: "sha256:stale"},
	}})
	if !errors.Is(err, content.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	var conflict *content.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected a *ConflictError, got %T", err)
	}
	if conflict.ID != item.ID {
		t.Errorf("conflict ID = %q, want %q", conflict.ID, item.ID)
	}
	if len(conflict.Theirs) == 0 {
		t.Error("conflict carries no on-disk bytes for a three-way diff")
	}
	if readFile(t, root, "content/posts/a/index.md") == "" {
		t.Error("file was damaged by a rejected write")
	}
}

func TestDuplicateIDIsHardError(t *testing.T) {
	root, types, _ := newTestProject(t)
	const fm = "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: %s\n---\nbody\n"
	writeFile(t, root, "content/posts/a/index.md", strings.Replace(fm, "%s", "a", 1))
	writeFile(t, root, "content/posts/a-copy/index.md", strings.Replace(fm, "%s", "a-copy", 1))

	_, err := NewScanner(root, types).Scan()
	if !errors.Is(err, content.ErrDuplicateID) {
		t.Fatalf("expected ErrDuplicateID, got %v", err)
	}
	if !strings.Contains(err.Error(), "content/posts/a/index.md") ||
		!strings.Contains(err.Error(), "content/posts/a-copy/index.md") {
		t.Errorf("error should name both files: %v", err)
	}
}

func TestDeleteBundleReportsEveryFile(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/a/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: a\n---\nbody\n")
	writeFile(t, root, "content/posts/a/cover.webp", "not really an image")

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.DeleteContent{ID: scan.Entries[0].Item.ID},
	}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	want := []string{"content/posts/a/cover.webp", "content/posts/a/index.md"}
	if !slices.Equal(res.Removed, want) {
		t.Errorf("Removed = %v, want %v", res.Removed, want)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "posts", "a")); !errors.Is(err, os.ErrNotExist) {
		t.Error("bundle directory still exists")
	}
}

func TestSoftDeleteKeepsFile(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/a/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: a\n---\nbody\n")

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.DeleteContent{ID: scan.Entries[0].Item.ID, Soft: true},
	}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got := readFile(t, root, "content/posts/a/index.md")
	if !strings.Contains(got, "deleted_at:") {
		t.Errorf("soft delete did not record deleted_at:\n%s", got)
	}
}

func TestPutMediaLandsInBundle(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/a/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: a\n---\nbody\n")

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.PutMedia{Owner: scan.Entries[0].Item.ID, Name: "cover.webp", Data: []byte("bytes")},
	}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Equal(res.Written, []string{"content/posts/a/cover.webp"}) {
		t.Errorf("Written = %v", res.Written)
	}
	if got := readFile(t, root, "content/posts/a/cover.webp"); got != "bytes" {
		t.Errorf("media content = %q", got)
	}
}

func TestMediaNameCannotEscapeBundle(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/a/index.md", "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\ntitle: A\nslug: a\n---\nbody\n")

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.PutMedia{Owner: scan.Entries[0].Item.ID, Name: "../../escaped.txt", Data: []byte("x")},
	}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Equal(res.Written, []string{"content/posts/a/escaped.txt"}) {
		t.Errorf("path traversal was not contained: %v", res.Written)
	}
	if _, err := os.Stat(filepath.Join(root, "escaped.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Error("file escaped the project root")
	}
}

func TestReadsHugoStyleFrontMatter(t *testing.T) {
	root, types, _ := newTestProject(t)
	writeFile(t, root, "content/posts/legacy/index.md", `---
title: Legacy Post
date: 2024-03-05
lastmod: 2024-04-01
draft: true
tags:
  - Go
categories: Tech
---
legacy body
`)

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(scan.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d (%v)", len(scan.Entries), scan.Problems)
	}
	item := scan.Entries[0].Item

	if item.Status != content.StatusDraft {
		t.Errorf("draft: true was not read as a draft status, got %q", item.Status)
	}
	if item.Slug != "legacy" {
		t.Errorf("slug should fall back to the directory name, got %q", item.Slug)
	}
	if want := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC); !item.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", item.CreatedAt, want)
	}
	if !slices.Equal(item.Terms("tags"), []string{"Go"}) {
		t.Errorf("tag = %v", item.Terms("tags"))
	}
	// A bare scalar is the common shorthand for a single term.
	if !slices.Equal(item.Terms("categories"), []string{"Tech"}) {
		t.Errorf("category = %v", item.Terms("categories"))
	}
	if item.ID != "" {
		t.Errorf("legacy file should have no ID until doctor assigns one, got %q", item.ID)
	}
	if len(scan.MissingIDs()) != 1 {
		t.Error("MissingIDs did not report the legacy file")
	}
}

func TestSinglePageLayout(t *testing.T) {
	root, _, w := newTestProject(t)
	item := &content.Content{
		Kind:   "page",
		Title:  "About",
		Status: content.StatusPublished,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: "\nabout\n"},
	}
	res, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Equal(res.Written, []string{"content/pages/about.md"}) {
		t.Errorf("Written = %v", res.Written)
	}
	_ = root
}

func TestBodyLayoutIsTheFiles(t *testing.T) {
	root, _, w := newTestProject(t)
	ctx := t.Context()

	// An editor hands over the text alone; the file gets its blank line and
	// its final newline from the store.
	item := &content.Content{
		Kind:   "post",
		Title:  "Layout",
		Status: content.StatusDraft,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: "first\n\nsecond"},
	}
	if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}}); err != nil {
		t.Fatal(err)
	}
	src := SourcePath(content.DefaultRegistry().Get("post"), item.Locator)
	first := readFile(t, root, src)
	if !strings.HasSuffix(first, "---\n\nfirst\n\nsecond\n") {
		t.Errorf("body not laid out:\n%s", first)
	}

	// What comes back is the text alone again, so saving it once more moves
	// nothing but updated_at.
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(src)))
	if err != nil {
		t.Fatal(err)
	}
	read, err := NewCodec(content.DefaultRegistry()).Decode(content.DefaultRegistry().Get("post"), item.Locator, data)
	if err != nil {
		t.Fatal(err)
	}
	if read.Body.Raw != "first\n\nsecond" {
		t.Errorf("Raw = %q", read.Body.Raw)
	}
	read.Kind = "post"
	if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{
		content.PutContent{Content: read, IfRevision: read.Revision},
	}}); err != nil {
		t.Fatal(err)
	}
	for _, line := range diffLines(first, readFile(t, root, src)) {
		if !strings.Contains(line, "updated_at") {
			t.Errorf("unexpected change on re-save: %q", line)
		}
	}
}

// The folder and the slug come from one key, so they cannot part company. A
// second item whose folder was same-title-2 while its slug stayed same-title
// would be written and then refused by the read model, which requires
// (kind, slug, locale) to be unique -- leaving a file nothing can load.
func TestSlugCollisionSuffixesFolderAndSlugTogether(t *testing.T) {
	root, _, w := newTestProject(t)
	ctx := t.Context()
	var locators []content.Locator
	var slugs []string
	for range 2 {
		item := &content.Content{
			Kind:   "post",
			Title:  "Same Title",
			Status: content.StatusDraft,
			Body:   content.Body{Format: content.FormatMarkdown, Raw: "x"},
		}
		if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}}); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		locators = append(locators, item.Locator)
		slugs = append(slugs, item.Slug)
	}
	want := []content.Locator{"content/posts/same-title", "content/posts/same-title-2"}
	if !slices.Equal(locators, want) {
		t.Errorf("locators = %v, want %v", locators, want)
	}
	if wantSlugs := []string{"same-title", "same-title-2"}; !slices.Equal(slugs, wantSlugs) {
		t.Errorf("slugs = %v, want %v", slugs, wantSlugs)
	}
	for _, loc := range want {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(string(loc)), BundleIndex)); err != nil {
			t.Errorf("%s was not created: %v", loc, err)
		}
	}
	got := readFile(t, root, "content/posts/same-title-2/"+BundleIndex)
	if !strings.Contains(got, "slug: same-title-2") {
		t.Errorf("the second file still claims the first one's slug:\n%s", got)
	}
}

// A slug the author typed is their URL, so a collision is refused instead of
// renamed -- and refused before any bytes land, because a file the index will
// not load is worse than a write that was turned down.
func TestATakenSlugIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	root, _, w := newTestProject(t)
	ctx := t.Context()

	first := &content.Content{Kind: "post", Title: "First", Slug: "notes", Status: content.StatusDraft}
	if _, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{content.PutContent{Content: first}}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	second := &content.Content{Kind: "post", Title: "Second", Slug: "notes", Status: content.StatusDraft}
	res, err := w.Apply(ctx, content.ChangeSet{Ops: []content.Op{content.PutContent{Content: second}}})
	if !errors.Is(err, content.ErrInvalid) {
		t.Fatalf("err = %v, want one wrapping ErrInvalid", err)
	}
	if !strings.Contains(err.Error(), "content/posts/notes") {
		t.Errorf("the error should name what holds the slug: %v", err)
	}
	if len(res.Written) != 0 {
		t.Errorf("Written = %v, want nothing", res.Written)
	}
	entries, err := os.ReadDir(filepath.Join(root, "content", "posts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("%d bundles on disk, want only the first: a refused write must leave no orphan", len(entries))
	}
}

// A duplicate that is already on disk, from a merge or a hand edit, must not
// lock the author out: refusing every save would leave no way to change the
// slug that is the problem.
func TestADuplicateAlreadyOnDiskDoesNotBlockAnEdit(t *testing.T) {
	root, types, w := newTestProject(t)
	writeFile(t, root, "content/posts/one/index.md",
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZAB1\ntitle: One\nslug: notes\nstatus: draft\n---\n\none\n")
	writeFile(t, root, "content/posts/two/index.md",
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZAB2\ntitle: Two\nslug: notes\nstatus: draft\n---\n\ntwo\n")

	item := &content.Content{
		ID:     "01J8KQ2P3R4S5T6V7W8X9YZAB1",
		Kind:   "post",
		Title:  "One",
		Slug:   "notes",
		Status: content.StatusDraft,
		Body:   content.Body{Format: content.FormatMarkdown, Raw: "\nedited\n"},
	}
	if _, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{content.PutContent{Content: item}}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := readFile(t, root, SourcePath(types.Get("post"), "content/posts/one")); !strings.Contains(got, "edited") {
		t.Errorf("the edit did not land:\n%s", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":      "hello-world",
		"  Trim  Me  ":     "trim-me",
		"Go 1.26 Released": "go-1-26-released",
		"你好，世界":            "你好-世界",
		"---":              "",
		"a---b":            "a-b",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// diffLines returns the lines present in exactly one of the two texts.
func diffLines(before, after string) []string {
	b := strings.Split(before, "\n")
	a := strings.Split(after, "\n")
	var out []string
	for _, line := range a {
		if !slices.Contains(b, line) {
			out = append(out, "+"+line)
		}
	}
	for _, line := range b {
		if !slices.Contains(a, line) {
			out = append(out, "-"+line)
		}
	}
	return out
}

// A Hugo site written in TOML has to open like one written in YAML: the same
// front matter in either language is the same item.
func TestATOMLPostReadsLikeItsYAMLTwin(t *testing.T) {
	root, types, _ := newTestProject(t)
	writeFile(t, root, "content/posts/in-yaml/index.md", `---
id: 01J8KQ2P3R4S5T6V7W8X9YZAAA
title: The Same Post
date: 2024-03-05T09:30:00+08:00
lastmod: 2024-04-01
draft: true
tags:
  - Go
  - Hugo
categories: Tech
weight: 3
description: Written twice.
---
The same body.
`)
	writeFile(t, root, "content/posts/in-toml/index.md", `+++
id = "01J8KQ2P3R4S5T6V7W8X9YZBBB"
title = "The Same Post"
date = 2024-03-05T09:30:00+08:00
lastmod = 2024-04-01
draft = true
tags = ["Go", "Hugo"]
categories = "Tech"
weight = 3
description = "Written twice."
+++
The same body.
`)

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(scan.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d (%v)", len(scan.Entries), scan.Problems)
	}
	items := map[string]*content.Content{}
	for _, e := range scan.Entries {
		items[e.Item.Slug] = e.Item
	}
	yml, tml := items["in-yaml"], items["in-toml"]
	if yml == nil || tml == nil {
		t.Fatalf("items = %v", items)
	}

	if tml.Title != yml.Title || tml.Status != yml.Status || tml.Body.Raw != yml.Body.Raw {
		t.Errorf("title, status or body differ:\n toml %q %q %q\n yaml %q %q %q",
			tml.Title, tml.Status, tml.Body.Raw, yml.Title, yml.Status, yml.Body.Raw)
	}
	if !tml.PublishedAt.Equal(*yml.PublishedAt) || !tml.UpdatedAt.Equal(yml.UpdatedAt) {
		t.Errorf("dates differ: toml %v %v, yaml %v %v", tml.PublishedAt, tml.UpdatedAt, yml.PublishedAt, yml.UpdatedAt)
	}
	for _, taxonomy := range []string{"tags", "categories"} {
		if !slices.Equal(tml.Terms(taxonomy), yml.Terms(taxonomy)) {
			t.Errorf("%s: toml %v, yaml %v", taxonomy, tml.Terms(taxonomy), yml.Terms(taxonomy))
		}
	}
	if fmt.Sprint(tml.Meta["weight"]) != fmt.Sprint(yml.Meta["weight"]) || tml.Meta["description"] != yml.Meta["description"] {
		t.Errorf("meta differs: toml %v, yaml %v", tml.Meta, yml.Meta)
	}
}

// Saving a TOML post keeps it TOML, and renaming it rewrites the title line
// and nothing the author wrote around it.
func TestEditingATOMLTitleRewritesOnlyTheTitleLine(t *testing.T) {
	root, types, w := newTestProject(t)
	const src = `+++
id = "01J8KQ2P3R4S5T6V7W8X9YZABC"
title = "Original Title" # shown in the feed too
slug = "original"
status = "published"
created_at = 2026-01-01T00:00:00Z
updated_at = 2026-01-01T00:00:00Z
tags = ["Go", "CMS"]

# for the docs site
[params]
  custom_key = "keep me"
+++

Body stays put.
`
	writeFile(t, root, "content/posts/original/index.md", src)

	scan, err := NewScanner(root, types).Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	item := scan.Entries[0].Item
	item.Title = "Renamed Title"
	if _, err := w.Apply(t.Context(), content.ChangeSet{Ops: []content.Op{
		content.PutContent{Content: item, IfRevision: item.Revision},
	}}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got := readFile(t, root, "content/posts/original/index.md")
	for _, line := range diffLines(src, got) {
		if !strings.Contains(line, "title = ") && !strings.Contains(line, "updated_at = ") {
			t.Errorf("unexpected line changed: %q", line)
		}
	}
	for _, must := range []string{
		`title = "Renamed Title" # shown in the feed too`,
		`tags = ["Go", "CMS"]`,
		"# for the docs site",
		`  custom_key = "keep me"`,
		"Body stays put.",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("lost %q:\n%s", must, got)
		}
	}
	if !strings.HasPrefix(got, "+++\n") {
		t.Errorf("the file is no longer TOML:\n%s", got)
	}
}
