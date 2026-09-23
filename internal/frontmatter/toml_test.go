package frontmatter

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// A post as Hugo writes one, with everything that makes TOML awkward to edit
// by line: a comment on a value, a date with and without a zone, a list over
// several lines, a string over several lines, and tables after the keys.
const hugoPost = `+++
title = "Hello World" # what the page is called
date = 2024-03-01T09:30:00+08:00
lastmod = 2024-03-02
draft = true
tags = [
  "Go",   # the language
  "CMS",
]
summary = """
Two lines
of summary."""
weight = 3

[params]
  cover = "cover.png"

[[resources]]
  src = "images/*.png"
+++

# Hello World

Body text stays exactly as it was.
`

func TestTOMLRoundTripsUntouched(t *testing.T) {
	d := mustParse(t, hugoPost)
	if d.Dirty() {
		t.Fatal("a document nobody edited is dirty")
	}
	if got := mustBytes(t, d); got != hugoPost {
		t.Errorf("round trip changed the document:\n%s", got)
	}
}

func TestTOMLValuesReadLikeYAMLOnes(t *testing.T) {
	d := mustParse(t, hugoPost)

	if got := d.Keys(); !slices.Equal(got, []string{"title", "date", "lastmod", "draft", "tags", "summary", "weight", "params", "resources"}) {
		t.Errorf("keys = %v", got)
	}
	if got := d.String("title"); got != "Hello World" {
		t.Errorf("title = %q", got)
	}
	if got := d.StringSlice("tags"); !slices.Equal(got, []string{"Go", "CMS"}) {
		t.Errorf("tags = %v", got)
	}
	if !d.Bool("draft") {
		t.Error("draft is not true")
	}
	if got, _ := d.Get("date"); !got.(time.Time).Equal(time.Date(2024, 3, 1, 1, 30, 0, 0, time.UTC)) {
		t.Errorf("date = %v", got)
	}
	// A date without a zone is read as UTC, as the index stores it.
	if got, _ := d.Get("lastmod"); !got.(time.Time).Equal(time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastmod = %v", got)
	}
	if got := d.String("summary"); got != "Two lines\nof summary." {
		t.Errorf("summary = %q", got)
	}
	if params, _ := d.Get("params"); params.(map[string]any)["cover"] != "cover.png" {
		t.Errorf("params = %v", params)
	}
	if got := d.Body(); !strings.HasPrefix(got, "\n# Hello World") {
		t.Errorf("body = %q", got)
	}
}

// Editing the title rewrites the title's line, keeps the note the author left
// on it, and touches nothing else.
func TestTOMLChangingOneKeyChangesOneLine(t *testing.T) {
	d := mustParse(t, hugoPost)
	if err := d.Set("title", "Hello, TOML"); err != nil {
		t.Fatal(err)
	}
	got := mustBytes(t, d)
	if diff := changedLines(hugoPost, got); !slices.Equal(diff, []string{
		`+title = "Hello, TOML" # what the page is called`,
		`-title = "Hello World" # what the page is called`,
	}) {
		t.Errorf("diff = %q", diff)
	}
}

func TestTOMLSettingWhatIsThereChangesNothing(t *testing.T) {
	d := mustParse(t, hugoPost)
	for key, value := range map[string]any{
		"title":   "Hello World",
		"date":    time.Date(2024, 3, 1, 1, 30, 0, 0, time.UTC),
		"lastmod": time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		"tags":    []string{"Go", "CMS"},
		"weight":  3,
		"params":  map[string]any{"cover": "cover.png"},
	} {
		if err := d.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if d.Dirty() {
		t.Error("setting every key to what it already was made the document dirty")
	}
}

// A new key goes after the last top level key: written after a table, it
// would belong to that table.
func TestTOMLNewKeysGoBeforeTheTables(t *testing.T) {
	d := mustParse(t, hugoPost)
	if err := d.SetAll([]string{"id", "status"}, map[string]any{
		"id": "01J8KQ2P3R4S5T6V7W8X9YZABC", "status": "published",
	}); err != nil {
		t.Fatal(err)
	}
	d.Delete("draft")
	got := mustBytes(t, d)

	want := strings.Replace(hugoPost, "draft = true\n", "", 1)
	want = strings.Replace(want, "weight = 3\n", "weight = 3\nid = \"01J8KQ2P3R4S5T6V7W8X9YZABC\"\nstatus = \"published\"\n", 1)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	again := mustParse(t, got)
	if again.String("id") != "01J8KQ2P3R4S5T6V7W8X9YZABC" || again.String("status") != "published" {
		t.Error("the new keys do not read back as top level keys")
	}
	if params, _ := again.Get("params"); params.(map[string]any)["cover"] != "cover.png" {
		t.Error("the table lost its keys")
	}
}

// A value over several lines is replaced whole, and a date keeps the form it
// was written in.
func TestTOMLReplacesAMultiLineValueAndKeepsADatesForm(t *testing.T) {
	d := mustParse(t, hugoPost)
	if err := d.Set("tags", []string{"Go", "CMS", "TOML"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Set("lastmod", time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	got := mustBytes(t, d)
	for _, want := range []string{`tags = ["Go", "CMS", "TOML"]` + "\n" + `summary = """`, "lastmod = 2024-04-05\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "# the language") {
		t.Error("the old list was left behind")
	}
	if again := mustParse(t, got); !slices.Equal(again.StringSlice("tags"), []string{"Go", "CMS", "TOML"}) {
		t.Errorf("tags read back as %v", again.StringSlice("tags"))
	}
}

// A table that changes is written again after the others; the rest stay.
func TestTOMLRewritesAChangedTable(t *testing.T) {
	d := mustParse(t, hugoPost)
	if err := d.Set("params", map[string]any{"cover": "new.png"}); err != nil {
		t.Fatal(err)
	}
	got := mustBytes(t, d)
	again := mustParse(t, got)
	if params, _ := again.Get("params"); params.(map[string]any)["cover"] != "new.png" {
		t.Errorf("params = %v in:\n%s", params, got)
	}
	if resources, _ := again.Get("resources"); len(resources.([]any)) != 1 {
		t.Errorf("the other table was lost:\n%s", got)
	}
	if !strings.Contains(got, "[[resources]]\n  src = \"images/*.png\"") {
		t.Errorf("the untouched table was rewritten:\n%s", got)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("moving the table left two blank lines together:\n%s", got)
	}
}

func TestTOMLKeepsLineEndingsAndAByteOrderMark(t *testing.T) {
	src := "\xEF\xBB\xBF" + strings.ReplaceAll("+++\ntitle = 'Hi'\n+++\nBody\n", "\n", "\r\n")
	d := mustParse(t, src)
	if err := d.Set("title", "Hello"); err != nil {
		t.Fatal(err)
	}
	// A literal string stays literal where the new value allows it.
	want := "\xEF\xBB\xBF" + strings.ReplaceAll("+++\ntitle = 'Hello'\n+++\nBody\n", "\n", "\r\n")
	if got := mustBytes(t, d); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTOMLQuotesWhatNeedsIt(t *testing.T) {
	d := mustParse(t, "+++\ntitle = \"x\"\n+++\n")
	title := "Say \"hi\"\\ now\ttab \x01"
	if err := d.Set("title", title); err != nil {
		t.Fatal(err)
	}
	if err := d.Set("with space", 1.5); err != nil {
		t.Fatal(err)
	}
	again := mustParse(t, mustBytes(t, d))
	if got := again.String("title"); got != title {
		t.Errorf("title read back as %q", got)
	}
	if got, _ := again.Get("with space"); got != 1.5 {
		t.Errorf("a key that needs quoting read back as %v", got)
	}
}

func TestTOMLPlacesDottedKeysWithQuotedParts(t *testing.T) {
	d := mustParse(t, "+++\ntitle = \"x\"\n\"a\" . 'b' = 1\n+++\n")
	if err := d.Set("title", "y"); err != nil {
		t.Fatal(err)
	}
	got := mustBytes(t, d)
	if got != "+++\ntitle = \"y\"\n\"a\" . 'b' = 1\n+++\n" {
		t.Errorf("got %q", got)
	}
	if a, _ := mustParse(t, got).Get("a"); a.(map[string]any)["b"] != int64(1) {
		t.Errorf("a = %v", a)
	}
}

// TOML this editor cannot place is still read; only an edit to it fails,
// rather than writing something it is not sure of.
func TestTOMLThatCannotBePlacedIsReadButNotEdited(t *testing.T) {
	d := mustParse(t, "+++\ntitle = \"x\"\n+++\n")
	d.toml.scanErr = errors.New("a construct the scanner does not know")

	if got := d.String("title"); got != "x" {
		t.Errorf("title = %q", got)
	}
	if got := mustBytes(t, d); got != "+++\ntitle = \"x\"\n+++\n" {
		t.Errorf("an unedited document did not round trip: %q", got)
	}
	if err := d.Set("title", "y"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Bytes(); err == nil {
		t.Error("an edit to TOML the editor could not place was written anyway")
	}
}

func TestTOMLWithoutAClosingDelimiterIsRefused(t *testing.T) {
	if _, err := Parse([]byte("+++\ntitle = \"x\"\n")); err == nil {
		t.Error("an unclosed front matter block parsed")
	}
}
