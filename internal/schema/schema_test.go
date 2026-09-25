package schema_test

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/schema"
)

func parse(t *testing.T, text string) schema.Schema {
	t.Helper()
	var s schema.Schema
	if err := yaml.Unmarshal([]byte(text), &s); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	return s
}

const settings = `
- key: look
  type: section
  label: Look
  fields:
    - {key: accent, type: color, default: "#7d5c3c"}
    - key: scheme
      type: select
      default: auto
      options: [{value: auto, label: Auto}, {value: dark, label: Dark}]
- key: show_toc
  type: boolean
  default: true
- key: columns
  type: number
  default: 2
- key: social
  type: group
  fields:
    - {key: github, type: url}
    - {key: mastodon, type: url, default: "https://example.social/@me"}
- key: nav
  type: repeat
  fields:
    - {key: label, type: string}
    - {key: url, type: url}
- key: langs
  type: multiselect
  options: [{value: en, label: English}, {value: zh, label: Chinese}]
`

// A section arranges the form and nothing else: the fields in it are stored
// where they would be without it, so a theme can add one to a form people
// have already filled in without moving a single value.
func TestASectionKeepsItsFieldsWhereTheyWere(t *testing.T) {
	s := parse(t, settings)

	if f := s.Field("accent"); f == nil || f.Type != schema.TypeColor {
		t.Errorf("Field(accent) = %v, want the color inside the section", f)
	}
	if f := s.Field("look"); f != nil {
		t.Errorf("Field(look) = %v, want nil: a section holds no value", f)
	}
	defaults := s.Defaults()
	if defaults["accent"] != "#7d5c3c" || defaults["scheme"] != "auto" {
		t.Errorf("defaults = %v, want the section's defaults at the top", defaults)
	}
	if _, ok := defaults["look"]; ok {
		t.Errorf("defaults = %v, want no value for the section", defaults)
	}
}

func TestASectionIsCheckedLikeTheLevelItSitsOn(t *testing.T) {
	for name, text := range map[string]string{
		"a key used twice across a section": `
- key: look
  type: section
  fields: [{key: accent, type: color}]
- {key: accent, type: string}`,
		"a section in a section": `
- key: look
  type: section
  fields:
    - key: inner
      type: section
      fields: [{key: accent, type: color}]`,
		"a section in a group": `
- key: social
  type: group
  fields:
    - key: inner
      type: section
      fields: [{key: github, type: url}]`,
		"a section with a default": `
- key: look
  type: section
  default: x
  fields: [{key: accent, type: color}]`,
		"an empty section": `
- key: look
  type: section`,
	} {
		var s schema.Schema
		if err := yaml.Unmarshal([]byte(text), &s); err != nil {
			t.Fatalf("%s: parse: %v", name, err)
		}
		if err := s.Validate(); err == nil {
			t.Errorf("%s: Validate accepted it", name)
		}
	}
}

// A theme whose default its own field would refuse is caught when the theme
// loads, rather than the first time someone saves the form.
func TestADefaultHasToFitItsField(t *testing.T) {
	for name, text := range map[string]string{
		"boolean": `[{key: x, type: boolean, default: "sometimes"}]`,
		"select":  `[{key: x, type: select, default: c, options: [{value: a, label: A}]}]`,
		"color":   `[{key: x, type: color, default: brown}]`,
		"number":  `[{key: x, type: number, default: 50, max: 10}]`,
	} {
		var s schema.Schema
		if err := yaml.Unmarshal([]byte(text), &s); err != nil {
			t.Fatalf("%s: parse: %v", name, err)
		}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "default") {
			t.Errorf("%s: Validate = %v, want the default refused", name, err)
		}
	}
}

// Resolve is what a template reads, so it makes the best of any file: a value
// that fits is kept, one that does not falls back to the default, and keys the
// theme never declared are left for whoever put them there.
func TestResolveReadsWhateverTheFileHolds(t *testing.T) {
	s := parse(t, settings)
	got := s.Resolve(map[string]any{
		"accent":   "#a3473b",
		"scheme":   "sepia", // not an option any more
		"show_toc": "false", // quoted by hand
		"columns":  "3",
		"social":   map[string]any{"github": "https://github.com/me"},
		"langs":    []any{"zh", "fr", "zh"},
		"extra":    "kept",
	})
	want := map[string]any{
		"accent":   "#a3473b",
		"scheme":   "auto",
		"show_toc": false,
		"columns":  float64(3),
		"social":   map[string]any{"github": "https://github.com/me", "mastodon": "https://example.social/@me"},
		"nav":      []any{},
		"langs":    []any{"zh"},
		"extra":    "kept",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve =\n  %v\nwant\n  %v", got, want)
	}
}

// A group is a map even when nothing is stored in it, so a template can write
// .social.github without first asking whether .social is there.
func TestAGroupIsAlwaysThere(t *testing.T) {
	s := parse(t, settings)
	got := s.Resolve(nil)
	if _, ok := got["social"].(map[string]any); !ok {
		t.Errorf("social = %#v, want a map", got["social"])
	}
	if got["columns"] != 2 {
		t.Errorf("columns = %#v, want the default as written", got["columns"])
	}
}

// A theme that asked for its menu as text, one "Label | /path/" a line, can
// ask for a list instead, and every site that filled the text in keeps its
// menu.
func TestARepeatReadsTheTextItReplaced(t *testing.T) {
	s := parse(t, settings)
	got := s.Resolve(map[string]any{"nav": "About | /about/\n\n Links|/links/ \n"})["nav"]
	want := []any{
		map[string]any{"label": "About", "url": "/about/"},
		map[string]any{"label": "Links", "url": "/links/"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nav = %v, want %v", got, want)
	}
}

func TestCheckRefusesWhatAFormShouldNotSend(t *testing.T) {
	s := parse(t, `
- {key: accent, type: color}
- {key: link, type: url}
- {key: count, type: number, min: 1, max: 10}
- {key: title, type: string, required: true}
- key: scheme
  type: select
  options: [{value: auto, label: Auto}]
- key: nav
  type: repeat
  fields:
    - {key: label, type: string, required: true}
    - {key: url, type: url}
`)
	for _, tc := range []struct {
		key     string
		value   any
		refused bool
	}{
		{"accent", "#7d5c3c", false},
		{"accent", "", false},
		{"accent", "brown", true},
		{"link", "https://example.com/x", false},
		{"link", "/about/", false},
		{"link", "mailto:me@example.com", false},
		{"link", "javascript:alert(1)", true},
		{"link", "https://", true},
		{"count", float64(5), false},
		{"count", float64(11), true},
		{"count", "5", true},
		{"title", "Hello", false},
		{"title", " ", true},
		{"scheme", "auto", false},
		{"scheme", "dark", true},
		{"nav", []any{map[string]any{"label": "About", "url": "/about/"}}, false},
		{"nav", []any{map[string]any{"url": "/about/"}}, true},
		{"nav", []any{map[string]any{"label": "About", "url": "javascript:x"}}, true},
		{"nav", "About | /about/", true},
	} {
		problem := s.Field(tc.key).Check(tc.value)
		if (problem != "") != tc.refused {
			t.Errorf("Check(%s = %#v) = %q, refused want %v", tc.key, tc.value, problem, tc.refused)
		}
	}
}

func TestLookupDescendsIntoGroupsOnly(t *testing.T) {
	s := parse(t, settings)
	if f := s.Lookup([]string{"social", "github"}); f == nil || f.Key != "github" {
		t.Errorf("Lookup(social.github) = %v", f)
	}
	if f := s.Lookup([]string{"accent"}); f == nil {
		t.Error("Lookup(accent) found nothing inside the section")
	}
	for _, path := range [][]string{{"nav", "label"}, {"accent", "x"}, {"missing"}, nil} {
		if f := s.Lookup(path); f != nil {
			t.Errorf("Lookup(%v) = %v, want nil", path, f)
		}
	}
}
