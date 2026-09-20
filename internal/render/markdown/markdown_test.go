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
	if doc.WordCount != 8 {
		t.Errorf("WordCount = %d, want 8", doc.WordCount)
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
