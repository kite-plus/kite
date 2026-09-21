package frontmatter_test

import (
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/frontmatter"
)

// A configuration file is full of comments explaining why a value is what it
// is. Changing one setting must not cost the author those comments, nor the
// order they chose to write things in.
func TestEditingAConfigFileKeepsItsCommentsAndOrder(t *testing.T) {
	const src = `# The site as visitors see it.
site:
  title: My Blog        # shown in the header
  baseURL: https://example.com
  language: en

# Rendering.
build:
  pageSize: 10
  minify: false
`
	doc, err := frontmatter.ParseYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}

	// An untouched document must come back byte for byte.
	same, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(same) != src {
		t.Errorf("an untouched file did not round trip:\n%s", same)
	}

	if _, ok := doc.Get("site"); !ok {
		t.Fatal("site is missing")
	}
	if err := doc.Set("theme", map[string]any{"name": "default"}); err != nil {
		t.Fatal(err)
	}

	out, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, want := range []string{
		"# The site as visitors see it.",
		"shown in the header",
		"# Rendering.",
		"theme:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the result lost %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "site:") > strings.Index(got, "build:") {
		t.Errorf("the keys were reordered:\n%s", got)
	}
}

// A configuration file is where people leave notes to themselves. Changing a
// value nested two levels down must cost them nothing: not the comment beside
// it, not the order they chose, not the sections they did not touch.
func TestChangingANestedValueLeavesEverythingElseAlone(t *testing.T) {
	const src = `# How the site presents itself.
site:
  title: My Blog        # shown in the header
  baseURL: https://example.com
  language: en

theme:
  name: default
  settings:
    # Brand color, picked to match the logo.
    primary_color: "#4a77d6"
    show_toc: true

build:
  pageSize: 10
`
	doc, err := frontmatter.ParseYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}

	if err := doc.SetNested([]string{"theme", "settings"}, "primary_color", "#ff0000"); err != nil {
		t.Fatal(err)
	}
	if err := doc.SetNested([]string{"site"}, "title", "Renamed"); err != nil {
		t.Fatal(err)
	}

	out, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	// The two values changed.
	for _, want := range []string{`primary_color: "#ff0000"`, "title: Renamed"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	// Everything else did not.
	for _, want := range []string{
		"# How the site presents itself.",
		"# shown in the header",
		"# Brand color, picked to match the logo.",
		"baseURL: https://example.com",
		"language: en",
		"show_toc: true",
		"pageSize: 10",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the edit destroyed %q:\n%s", want, got)
		}
	}
	// And the order is the one the author wrote.
	for _, pair := range [][2]string{
		{"site:", "theme:"},
		{"theme:", "build:"},
		{"title:", "baseURL:"},
		{"primary_color:", "show_toc:"},
	} {
		if strings.Index(got, pair[0]) > strings.Index(got, pair[1]) {
			t.Errorf("%s and %s were reordered:\n%s", pair[0], pair[1], got)
		}
	}
	// The double quotes the author used are still there.
	if strings.Contains(got, "primary_color: #ff0000") {
		t.Errorf("the quoting style was dropped:\n%s", got)
	}
}

// Setting something under a section that does not exist has nothing to
// preserve, so it is simply created.
func TestSettingUnderAMissingSectionCreatesIt(t *testing.T) {
	doc, err := frontmatter.ParseYAML([]byte("site:\n  title: Blog\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetNested([]string{"theme", "settings"}, "accent", "blue"); err != nil {
		t.Fatal(err)
	}
	out, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"theme:", "settings:", "accent: blue", "title: Blog"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// Setting a value to what it already is must leave the file untouched, or
// opening a settings page and closing it again would show up as a change.
func TestSettingTheSameValueChangesNothing(t *testing.T) {
	const src = "site:\n  title: Blog   # unchanged\n\ntheme:\n  settings:\n    accent: blue\n"
	doc, err := frontmatter.ParseYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetNested([]string{"theme", "settings"}, "accent", "blue"); err != nil {
		t.Fatal(err)
	}
	if doc.Dirty() {
		t.Error("writing back the same value marked the document changed")
	}
	out, err := doc.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != src {
		t.Errorf("the file changed:\n%s", out)
	}
}
