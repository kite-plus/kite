package index

import (
	"strings"
	"testing"
)

// A summary is read on its own, away from the page it came from, so none of
// the markup that gave the source its structure may survive into it.
func TestPlainProseDropsMarkup(t *testing.T) {
	const body = `A derived index has exactly one job.

## The failure nobody sees

The dangerous state is a **partially stale** index that *looks* fine.

| | Build | Serve |
|---|---|---|
| Files | a generator | a preview server |

` + "```go" + `
func main() { fmt.Println("not prose") }
` + "```" + `

- a bullet
- another one

See [the design notes](docs/design/architecture.md) for the reasoning.

> A quoted aside.

---

![a diagram](diagram.png)
`

	got := plainProse(body)

	for _, unwanted := range []string{
		"##", "|", "```", "func main", "**", "*looks*",
		"docs/design", "diagram.png", ">", "---", "- a bullet",
	} {
		if strings.Contains(got, unwanted) {
			t.Errorf("summary kept %q:\n%s", unwanted, got)
		}
	}
	for _, wanted := range []string{
		"A derived index has exactly one job.",
		"partially stale",
		"the design notes",
		"a bullet",
	} {
		if !strings.Contains(got, wanted) {
			t.Errorf("summary lost %q:\n%s", wanted, got)
		}
	}
}

func TestDescriptionIsPlainText(t *testing.T) {
	got := summarize("A cache that is only *usually* right is **wrong**.", "body")
	if strings.ContainsAny(got, "*_`") {
		t.Errorf("summary kept emphasis markers: %q", got)
	}
	if got != "A cache that is only usually right is wrong." {
		t.Errorf("summarize = %q", got)
	}
}

func TestSummarizePrefersAuthoredDescription(t *testing.T) {
	const body = "The opening sentence of the article body.\n"
	const desc = "A sentence written to stand on its own."

	if got := summarize(desc, body); got != desc {
		t.Errorf("summarize = %q, want the description %q", got, desc)
	}
	if got := summarize("   ", body); !strings.HasPrefix(got, "The opening sentence") {
		t.Errorf("a blank description should fall through to the body, got %q", got)
	}
}

func TestTruncateEndsOnAWord(t *testing.T) {
	long := strings.Repeat("alpha beta ", 60)
	got := summarize("", long)

	if len([]rune(got)) > excerptLimit+1 {
		t.Errorf("summary is %d runes, want at most %d", len([]rune(got)), excerptLimit+1)
	}
	if !strings.HasSuffix(got, "\u2026") {
		t.Errorf("a truncated summary should end with an ellipsis: %q", got)
	}
	// Cutting mid-word is the thing a limit is supposed to avoid.
	trimmed := strings.TrimSuffix(got, "\u2026")
	if !strings.HasSuffix(trimmed, "alpha") && !strings.HasSuffix(trimmed, "beta") {
		t.Errorf("summary ends mid-word: %q", got)
	}
}

func TestShortBodyIsNotTruncated(t *testing.T) {
	got := summarize("", "One short line.\n")
	if got != "One short line." {
		t.Errorf("summarize = %q", got)
	}
}

func TestInlineText(t *testing.T) {
	cases := map[string]string{
		"plain words":                   "plain words",
		"**bold** and *italic*":         "bold and italic",
		"`code` inline":                 "code inline",
		"a [link](https://example.com)": "a link",
		"![img](x.png) after":           "after",
		"1. numbered item":              "numbered item",
		"- bulleted item":               "bulleted item",
		"~~struck~~ through":            "struck through",
		"[bare brackets]":               "bare brackets",
	}
	for in, want := range cases {
		if got := inlineText(in); got != want {
			t.Errorf("inlineText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSkipLine(t *testing.T) {
	for line, want := range map[string]bool{
		"":                  true,
		"## heading":        true,
		"| a | b |":         true,
		"> quote":           true,
		"---":               true,
		"***":               true,
		"<div>":             true,
		"ordinary prose":    false,
		"- a list item":     false,
		"a - b in a phrase": false,
	} {
		if got := skipLine(line); got != want {
			t.Errorf("skipLine(%q) = %v, want %v", line, got, want)
		}
	}
}
