package frontmatter

import (
	"slices"
	"strings"
	"testing"
)

const sample = `---
# the item identity, do not edit
id: 01J8KQ2P3R4S5T6V7W8X9YZABC
title: Hello World
slug: hello-world
status: draft
tags: [Go, CMS]
categories:
  - Tech
  - Notes
description: >-
  A folded
  description.
pinned: false
---

# Hello World

Body text stays exactly as it was.
`

func mustParse(t *testing.T, src string) *Document {
	t.Helper()
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func mustBytes(t *testing.T, d *Document) string {
	t.Helper()
	b, err := d.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	return string(b)
}

// changedLines reports the lines that differ between two documents, which is
// the property the whole package exists to control.
func changedLines(before, after string) []string {
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

func TestRoundTripIsByteIdentical(t *testing.T) {
	cases := map[string]string{
		"full":           sample,
		"no frontmatter": "# Just a body\n\nNo front matter here.\n",
		"empty fm":       "---\n---\nbody\n",
		"no trailing nl": "---\ntitle: X\n---\nbody",
		"crlf":           "---\r\ntitle: X\r\n---\r\nbody\r\n",
		"bom":            "\ufeff---\ntitle: X\n---\nbody\n",
		"dots close":     "---\ntitle: X\n...\nbody\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			d := mustParse(t, src)
			if d.Dirty() {
				t.Fatal("freshly parsed document reports dirty")
			}
			if got := mustBytes(t, d); got != src {
				t.Errorf("round trip changed bytes\n got: %q\nwant: %q", got, src)
			}
		})
	}
}

func TestSetIdenticalValueIsNoop(t *testing.T) {
	d := mustParse(t, sample)
	for _, tc := range []struct {
		key string
		val any
	}{
		{"title", "Hello World"},
		{"slug", "hello-world"},
		{"pinned", false},
		{"tags", []string{"Go", "CMS"}},
		{"categories", []string{"Tech", "Notes"}},
	} {
		if err := d.Set(tc.key, tc.val); err != nil {
			t.Fatalf("Set(%q): %v", tc.key, err)
		}
	}
	if d.Dirty() {
		t.Fatal("setting identical values marked the document dirty")
	}
	if got := mustBytes(t, d); got != sample {
		t.Errorf("no-op writes changed bytes:\n%s", strings.Join(changedLines(sample, got), "\n"))
	}
}

func TestChangingTitleTouchesOnlyTitleLine(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("title", "Hello Kite"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)

	diff := changedLines(sample, got)
	want := []string{"+title: Hello Kite", "-title: Hello World"}
	slices.Sort(diff)
	slices.Sort(want)
	if !slices.Equal(diff, want) {
		t.Errorf("expected exactly the title line to change, got:\n%s", strings.Join(diff, "\n"))
	}
}

func TestUnrelatedEditPreservesFlowStyleAndComments(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("status", "published"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)

	for _, must := range []string{
		"# the item identity, do not edit",
		"tags: [Go, CMS]",
		"categories:\n  - Tech\n  - Notes",
		"description: >-",
		"status: published",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("output lost %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "status: draft") {
		t.Error("old status value still present")
	}
}

func TestFlowSequenceStaysFlowWhenEdited(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("tags", []string{"Go", "CMS", "Static"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)
	if !strings.Contains(got, "tags: [Go, CMS, Static]") {
		t.Errorf("flow sequence was reflowed as a block list:\n%s", got)
	}
}

func TestBlockSequenceStaysBlockWhenEdited(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("categories", []string{"Tech", "Notes", "Go"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)
	if !strings.Contains(got, "categories:\n  - Tech\n  - Notes\n  - Go") {
		t.Errorf("block sequence was reflowed:\n%s", got)
	}
}

func TestKeyOrderAndBodyPreserved(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("status", "published"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)

	after := mustParse(t, got)
	if !slices.Equal(after.Keys(), d.Keys()) {
		t.Errorf("key order changed: %v -> %v", d.Keys(), after.Keys())
	}
	if after.Body() != d.Body() {
		t.Errorf("body changed:\n got: %q\nwant: %q", after.Body(), d.Body())
	}
}

func TestInsertAppendsAtEnd(t *testing.T) {
	d := mustParse(t, sample)
	if err := d.Set("updated_at", "2026-09-21T00:00:00Z"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)

	after := mustParse(t, got)
	keys := after.Keys()
	if keys[len(keys)-1] != "updated_at" {
		t.Errorf("new key not appended last: %v", keys)
	}
	if v := after.String("updated_at"); v != "2026-09-21T00:00:00Z" {
		t.Errorf("updated_at = %q", v)
	}
	// Every pre-existing key keeps its position.
	if !slices.Equal(keys[:len(keys)-1], d.Keys()) {
		t.Errorf("existing key order disturbed: %v", keys)
	}
}

func TestDelete(t *testing.T) {
	d := mustParse(t, sample)
	d.Delete("pinned")
	got := mustBytes(t, d)
	if strings.Contains(got, "pinned") {
		t.Errorf("deleted key still present:\n%s", got)
	}
	after := mustParse(t, got)
	if slices.Contains(after.Keys(), "pinned") {
		t.Error("deleted key still parses")
	}
	if !strings.Contains(got, "tags: [Go, CMS]") {
		t.Error("delete disturbed a neighboring key")
	}
}

func TestDeleteAbsentKeyIsNoop(t *testing.T) {
	d := mustParse(t, sample)
	d.Delete("nope")
	if d.Dirty() {
		t.Fatal("deleting an absent key marked the document dirty")
	}
}

func TestCreateFrontMatterWhenMissing(t *testing.T) {
	const src = "# Title\n\nbody\n"
	d := mustParse(t, src)
	if err := d.Set("id", "01J8KQ2P3R4S5T6V7W8X9YZABC"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)
	if !strings.HasPrefix(got, "---\nid: 01J8KQ2P3R4S5T6V7W8X9YZABC\n---\n") {
		t.Errorf("front matter not created:\n%s", got)
	}
	if !strings.HasSuffix(got, src) {
		t.Errorf("body not preserved:\n%s", got)
	}
}

func TestBlockScalarContainingHashIsNotTreatedAsComment(t *testing.T) {
	const src = `---
title: X
script: |
  echo one
  # a comment inside the script
slug: x
---
body
`
	d := mustParse(t, src)
	if err := d.Set("title", "Y"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)
	if !strings.Contains(got, "  # a comment inside the script") {
		t.Errorf("indented comment inside block scalar was lost:\n%s", got)
	}
	if !strings.Contains(got, "slug: x") {
		t.Errorf("following key was lost:\n%s", got)
	}
}

func TestCommentBelongingToNextKeySurvivesEdit(t *testing.T) {
	const src = `---
title: X

# this comment introduces slug
slug: x
---
body
`
	d := mustParse(t, src)
	if err := d.Set("title", "Y"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := mustBytes(t, d)
	if strings.Count(got, "# this comment introduces slug") != 1 {
		t.Errorf("head comment duplicated or lost:\n%s", got)
	}
	if !strings.Contains(got, "title: Y") {
		t.Errorf("title not updated:\n%s", got)
	}
}

func TestSetBodyKeepsFrontMatterBytes(t *testing.T) {
	d := mustParse(t, sample)
	d.SetBody("\nnew body\n")
	got := mustBytes(t, d)
	if !strings.Contains(got, "tags: [Go, CMS]") || !strings.Contains(got, "# the item identity, do not edit") {
		t.Errorf("front matter disturbed by a body-only edit:\n%s", got)
	}
	if !strings.HasSuffix(got, "\nnew body\n") {
		t.Errorf("body not replaced:\n%s", got)
	}
}

func TestSetSameBodyIsNoop(t *testing.T) {
	d := mustParse(t, sample)
	d.SetBody(d.Body())
	if d.Dirty() {
		t.Fatal("setting an identical body marked the document dirty")
	}
}

func TestTypedGetters(t *testing.T) {
	d := mustParse(t, sample)
	if got := d.String("slug"); got != "hello-world" {
		t.Errorf("String(slug) = %q", got)
	}
	if got := d.StringSlice("tags"); !slices.Equal(got, []string{"Go", "CMS"}) {
		t.Errorf("StringSlice(tags) = %v", got)
	}
	if d.Bool("pinned") {
		t.Error("Bool(pinned) = true, want false")
	}
	if _, ok := d.Get("missing"); ok {
		t.Error("Get(missing) reported present")
	}
}

func TestUnterminatedFrontMatterIsAnError(t *testing.T) {
	if _, err := Parse([]byte("---\ntitle: X\nno closing delimiter\n")); err == nil {
		t.Fatal("expected an error for unterminated front matter")
	}
}

func TestNonMappingFrontMatterIsAnError(t *testing.T) {
	if _, err := Parse([]byte("---\n- just\n- a list\n---\nbody\n")); err == nil {
		t.Fatal("expected an error for non-mapping front matter")
	}
}
