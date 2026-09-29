package index

import (
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/render/markdown"
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

	got := summarize("", body).Excerpt

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
	got := summarize("A cache that is only *usually* right is **wrong**.", "body").Excerpt
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

	if got := summarize(desc, body).Excerpt; got != desc {
		t.Errorf("summarize = %q, want the description %q", got, desc)
	}
	if got := summarize("   ", body).Excerpt; !strings.HasPrefix(got, "The opening sentence") {
		t.Errorf("a blank description should fall through to the body, got %q", got)
	}
}

func TestTruncateEndsOnAWord(t *testing.T) {
	long := strings.Repeat("alpha beta ", 60)
	got := summarize("", long).Excerpt

	if len([]rune(got)) > markdown.ExcerptLimit+1 {
		t.Errorf("summary is %d runes, want at most %d", len([]rune(got)), markdown.ExcerptLimit+1)
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
	got := summarize("", "One short line.\n").Excerpt
	if got != "One short line." {
		t.Errorf("summarize = %q", got)
	}
}

// Markup goes and text stays, however it sits in a line: an underscore inside
// a word is not emphasis, and a paragraph may open with a link.
func TestSummaryKeepsTheText(t *testing.T) {
	cases := map[string]string{
		"plain words":                           "plain words",
		"**bold** and *italic*":                 "bold and italic",
		"`code` inline":                         "code inline",
		"a [link](https://example.com)":         "a link",
		"![img](x.png) after":                   "after",
		"1. numbered item":                      "numbered item",
		"- bulleted item":                       "bulleted item",
		"~~struck~~ through":                    "struck through",
		"[bare brackets]":                       "[bare brackets]",
		"[Kite](https://example.com) is a tool": "Kite is a tool",
		"a hit costs t_h and max_retries":       "a hit costs t_h and max_retries",
		`an escaped \*star\*`:                   "an escaped *star*",
	}
	for in, want := range cases {
		if got := summarize("", in).Excerpt; got != want {
			t.Errorf("summarize(%q).Excerpt = %q, want %q", in, got, want)
		}
	}
}

// A list says how long an item is by what the index counted, which is what
// the item's own page counts. A description is not part of the body.
func TestSummarizeCountsTheBody(t *testing.T) {
	const body = "## 桂花\n\nThe osmanthus bloomed a week early.\n\n```\nnot read\n```\n"
	got := summarize("A description of many more words than the body has.", body)
	if got.WordCount != 8 || got.CJKCount != 2 {
		t.Errorf("WordCount, CJKCount = %d, %d, want 8, 2", got.WordCount, got.CJKCount)
	}
	page, err := markdown.New(markdown.DefaultOptions()).Render(body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.WordCount != page.WordCount || got.CJKCount != page.CJKCount {
		t.Errorf("the index counts %d, %d, the page %d, %d", got.WordCount, got.CJKCount, page.WordCount, page.CJKCount)
	}
}

// A list may show one of an item's pictures, so the index keeps the pictures
// its body shows, in order and as written. A description is not the body.
func TestSummarizeListsThePictures(t *testing.T) {
	got := summarize("Written with a picture ![d](d.png).", "![a](a.png)\n\nText ![b](/uploads/b.jpg)\n")
	if want := []string{"a.png", "/uploads/b.jpg"}; !slices.Equal(got.Images, want) {
		t.Errorf("Images = %q, want %q", got.Images, want)
	}
}
