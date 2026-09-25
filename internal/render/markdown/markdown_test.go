package markdown_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/render/markdown"
)

func render(t *testing.T, src string, mutate func(*markdown.Options)) *markdown.Document {
	t.Helper()
	opts := markdown.DefaultOptions()
	if mutate != nil {
		mutate(&opts)
	}
	doc, err := markdown.New(opts).Render(src)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return doc
}

func TestRendersGFM(t *testing.T) {
	doc := render(t, `# Title

Some **bold** text with a [link](/posts/other/).

| a | b |
|---|---|
| 1 | 2 |

- [x] done
- [ ] pending

~~struck~~
`, nil)

	for _, want := range []string{"<h1", "<strong>bold</strong>", "<table>", "<del>struck</del>", `type="checkbox"`} {
		if !strings.Contains(doc.HTML, want) {
			t.Errorf("output missing %q:\n%s", want, doc.HTML)
		}
	}
}

// Raw HTML passing through is a site-wide decision that changes how every
// existing document renders, so the default has to be the conservative one.
func TestRawHTMLIsBlockedByDefault(t *testing.T) {
	const src = "Before\n\n<script>alert(1)</script>\n\nAfter\n"

	safe := render(t, src, nil)
	if strings.Contains(safe.HTML, "<script>") {
		t.Errorf("raw HTML leaked with the default options:\n%s", safe.HTML)
	}

	unsafe := render(t, src, func(o *markdown.Options) { o.UnsafeHTML = true })
	if !strings.Contains(unsafe.HTML, "<script>") {
		t.Errorf("UnsafeHTML did not allow raw HTML through:\n%s", unsafe.HTML)
	}
}

func TestCollectsTableOfContents(t *testing.T) {
	doc := render(t, `# One

text

## Two

more

### Three
`, nil)

	if len(doc.TOC) != 3 {
		t.Fatalf("got %d headings, want 3: %+v", len(doc.TOC), doc.TOC)
	}
	levels := []int{doc.TOC[0].Level, doc.TOC[1].Level, doc.TOC[2].Level}
	if !slices.Equal(levels, []int{1, 2, 3}) {
		t.Errorf("levels = %v", levels)
	}
	if doc.TOC[1].Text != "Two" {
		t.Errorf("heading text = %q", doc.TOC[1].Text)
	}
	if doc.TOC[1].ID == "" {
		t.Error("headings should carry generated anchors")
	}
	if !strings.Contains(doc.HTML, `id="`+doc.TOC[1].ID+`"`) {
		t.Error("the collected anchor does not match the rendered HTML")
	}
}

func TestCollectsLinksAndImages(t *testing.T) {
	doc := render(t, `[a](/one/) and ![img](cover.webp)

<https://example.com>
`, nil)

	if !slices.Contains(doc.Links, "/one/") {
		t.Errorf("links = %v", doc.Links)
	}
	if !slices.Contains(doc.Links, "https://example.com") {
		t.Errorf("autolink not collected: %v", doc.Links)
	}
	// Relative image references are what a page bundle relies on, and the
	// media reference tracker consumes this list.
	if !slices.Contains(doc.Images, "cover.webp") {
		t.Errorf("images = %v", doc.Images)
	}
}

func TestExcerptAndWordCount(t *testing.T) {
	doc := render(t, `# Heading

First paragraph with five words.

Second paragraph here.
`, nil)

	if doc.Excerpt != "First paragraph with five words." {
		t.Errorf("Excerpt = %q", doc.Excerpt)
	}
	// The heading is read as well as the two paragraphs.
	if doc.WordCount != 9 {
		t.Errorf("WordCount = %d, want 9", doc.WordCount)
	}
}

// A word count is of what a reader reads, and lists, headings and tables are
// read like any paragraph. A tight list keeps its items' text outside of
// paragraphs, so counting paragraphs alone called a page of links empty.
func TestWordCountReadsEveryKindOfText(t *testing.T) {
	for _, tc := range []struct {
		name  string
		src   string
		words int
	}{
		{"tight list", "- one two\n- three\n", 3},
		{"loose list", "- one two\n\n- three\n", 3},
		{"heading", "## Two words\n", 2},
		{"table", "| a | b |\n|---|---|\n| c d | e |\n", 5},
		{"quote", "> quoted words here\n", 3},
		{"definition", "Term\n: Its description\n", 3},
		{"footnote", "Text[^1].\n\n[^1]: A note.\n", 3},
		{"inline code and links", "Run `kite build` from [the root](/docs/) or https://example.com/x.\n", 8},
	} {
		if got := render(t, tc.src, nil).WordCount; got != tc.words {
			t.Errorf("%s: WordCount = %d, want %d", tc.name, got, tc.words)
		}
	}
}

// Code is skimmed rather than read, raw HTML is markup, and a picture is
// looked at; none of them makes a page longer to read.
func TestWordCountLeavesOutCodeAndPictures(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"fenced code", "Two words\n\n```go\nfunc main() { fmt.Println(\"hi\") }\n```\n"},
		{"indented code", "Two words\n\n    kite build --verify\n"},
		{"image", "![a cherry tree in bloom](cherry.jpg)\n\nTwo words\n"},
		{"html block", "<div>\nhidden words\n</div>\n\nTwo words\n"},
	} {
		if got := render(t, tc.src, nil).WordCount; got != 2 {
			t.Errorf("%s: WordCount = %d, want 2", tc.name, got)
		}
	}
}

// A Chinese paragraph has no spaces, so counting fields called a whole article
// a handful of words and every post a one minute read.
func TestWordCountCountsCJKCharacters(t *testing.T) {
	for _, tc := range []struct {
		name       string
		src        string
		words, cjk int
	}{
		{"chinese", "湖边的桂花开了，比去年早了差不多一周。", 17, 17},
		{"mixed", "用 kite build 一条命令就能重新发布。", 13, 11},
		{"japanese", "これはテストです。", 8, 8},
		{"contraction", "Don't count the apostrophe, or the dash - at all.", 9, 0},
	} {
		doc := render(t, tc.src, nil)
		if doc.WordCount != tc.words || doc.CJKCount != tc.cjk {
			t.Errorf("%s: WordCount, CJKCount = %d, %d, want %d, %d",
				tc.name, doc.WordCount, doc.CJKCount, tc.words, tc.cjk)
		}
	}
}

func TestExcerptIsTruncated(t *testing.T) {
	doc := render(t, strings.Repeat("word ", 200), nil)
	if len([]rune(doc.Excerpt)) > 201 {
		t.Errorf("excerpt not truncated: %d runes", len([]rune(doc.Excerpt)))
	}
	if !strings.HasSuffix(doc.Excerpt, "\u2026") {
		t.Errorf("truncated excerpt should end with an ellipsis: %q", doc.Excerpt)
	}
}

func TestCodeHighlighting(t *testing.T) {
	doc := render(t, "```go\nfunc main() {}\n```\n", nil)
	if !strings.Contains(doc.HTML, "<pre") {
		t.Errorf("code block not rendered:\n%s", doc.HTML)
	}
	if !strings.Contains(doc.HTML, "span") {
		t.Errorf("code was not highlighted:\n%s", doc.HTML)
	}
}

func TestFootnotes(t *testing.T) {
	doc := render(t, "Text with a note.[^1]\n\n[^1]: The note.\n", nil)
	if !strings.Contains(doc.HTML, "footnote") {
		t.Errorf("footnotes not rendered:\n%s", doc.HTML)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	const src = "# A\n\ntext [l](/x/) ![i](y.png)\n\n## B\n"
	r := markdown.New(markdown.DefaultOptions())
	first, err := r.Render(src)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		next, err := r.Render(src)
		if err != nil {
			t.Fatal(err)
		}
		if next.HTML != first.HTML {
			t.Fatal("repeated renders produced different HTML")
		}
		if !slices.Equal(next.Links, first.Links) || len(next.TOC) != len(first.TOC) {
			t.Fatal("repeated renders produced different metadata")
		}
	}
}

func TestEmptyBody(t *testing.T) {
	doc := render(t, "", nil)
	if doc.HTML != "" {
		t.Errorf("HTML = %q, want empty", doc.HTML)
	}
	if doc.WordCount != 0 || doc.Excerpt != "" {
		t.Errorf("unexpected metadata: %+v", doc)
	}
}
