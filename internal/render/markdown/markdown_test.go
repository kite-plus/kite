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

	if doc.Excerpt != "First paragraph with five words. Second paragraph here." {
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

// The text is what the word count counts, so a search finds a page by what
// its reader reads and not by its code or its markup.
func TestTextIsWhatAReaderReads(t *testing.T) {
	src := "# Cherry trees\n\nThey bloom in *April*.\n\n```go\nfmt.Println(\"hidden\")\n```\n\n" +
		"- one\n- two\n\n![a picture](tree.jpg)\n\n| a | b |\n|---|---|\n| c | d |\n"
	want := "Cherry trees\nThey bloom in April.\none\ntwo\na\nb\nc\nd"
	if got := render(t, src, nil).Text; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
}

// Escapes and references are written in the source for the renderer, and a
// reader of the excerpt or the text sees what they stand for.
func TestPlainTextReadsEscapesAndReferences(t *testing.T) {
	doc := render(t, "1\\. Not a list, a\\_b \\*c\\* &amp; &#169; \\&amp; `a\\_b`\n", nil)
	want := "1. Not a list, a_b *c* & \u00a9 &amp; a\\_b"
	if doc.Excerpt != want || doc.Text != want {
		t.Errorf("Excerpt = %q, Text = %q, want %q", doc.Excerpt, doc.Text, want)
	}
}

// A typographer writes its quotes and dashes as entities, which a reader of
// the excerpt, a heading or the text sees as the characters they stand for.
// Read as text, their names were counted as words.
func TestTypographerKeepsPlainText(t *testing.T) {
	src := "## Don't panic\n\nIt's here -- \"quoted\" ...\n"
	doc := render(t, src, func(o *markdown.Options) { o.Typographer = true })
	if want := "It\u2019s here \u2013 \u201cquoted\u201d \u2026"; doc.Excerpt != want {
		t.Errorf("Excerpt = %q, want %q", doc.Excerpt, want)
	}
	if want := "Don\u2019t panic"; len(doc.TOC) != 1 || doc.TOC[0].Text != want {
		t.Errorf("TOC = %+v, want one heading %q", doc.TOC, want)
	}
	if plain := render(t, src, nil); doc.WordCount != plain.WordCount {
		t.Errorf("WordCount = %d, want %d as without the typographer", doc.WordCount, plain.WordCount)
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

func TestExcerptIsTruncatedAtAWord(t *testing.T) {
	doc := render(t, strings.Repeat("alpha beta ", 60), nil)
	if n := len([]rune(doc.Excerpt)); n > markdown.ExcerptLimit+1 {
		t.Errorf("excerpt is %d runes, want at most %d", n, markdown.ExcerptLimit+1)
	}
	cut := strings.TrimSuffix(doc.Excerpt, "\u2026")
	if cut == doc.Excerpt {
		t.Errorf("a truncated excerpt should end with an ellipsis: %q", doc.Excerpt)
	}
	if !strings.HasSuffix(cut, "alpha") && !strings.HasSuffix(cut, "beta") {
		t.Errorf("excerpt ends mid-word: %q", doc.Excerpt)
	}

	// Math a plugin has written between \( and \) is not cut in two, which
	// would show its source.
	math := render(t, strings.Repeat("word ", 40)+`\\(a_1 + b_1 + c_1 + d_1 + e_1 + f_1 + g_1\\) end`, nil).Excerpt
	if strings.Contains(math, `\(`) || !strings.HasSuffix(math, "word\u2026") {
		t.Errorf("the excerpt cuts into math: %q", math)
	}

	// Chinese has no spaces to cut at, and is cut at the limit.
	zh := render(t, strings.Repeat("中文没有空格", 60), nil).Excerpt
	if n := len([]rune(zh)); n != markdown.ExcerptLimit+1 {
		t.Errorf("a Chinese excerpt is %d runes, want %d", n, markdown.ExcerptLimit+1)
	}
}

// An excerpt is the prose a body opens with. It is read from the document
// rather than scraped from its lines, so what is structure stays out and
// what is text stays in, however it is written.
func TestExcerptIsTheOpeningProse(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"structure left out",
			"# Title\n\nFirst words.\n\n> A quote.\n\n| a | b |\n|---|---|\n| c | d |\n\n```go\ncode()\n```\n\n<div>html</div>\n\n- one\n- two\n\nLast words.[^1]\n\n[^1]: A footnote.\n",
			"First words. one two Last words."},
		{"underscores inside words", "Set max_retries and snake_case keys.", "Set max_retries and snake_case keys."},
		{"escapes and entities", `1\. Not a list, a \*star\* &amp; more.`, "1. Not a list, a *star* & more."},
		{"a paragraph that opens with a link", "[Kite](https://example.com) builds sites.", "Kite builds sites."},
		{"a lone asterisk", "Two * three = six.", "Two * three = six."},
		{"a picture", "![a tree](tree.jpg)\n\nWords after it.", "Words after it."},
		{"a reference link defined later", "See [the notes][n].\n\nMore.\n\n[n]: https://example.com\n", "See the notes. More."},
	} {
		if got := render(t, tc.src, nil).Excerpt; got != tc.want {
			t.Errorf("%s: Excerpt = %q, want %q", tc.name, got, tc.want)
		}
		if got := markdown.Excerpt(tc.src); got != tc.want {
			t.Errorf("%s: Excerpt(source) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A list shows what the index read of a source without rendering it, and
// has to say what the item's own page says.
func TestSkimReadsAsRenderDoes(t *testing.T) {
	for _, src := range []string{
		"# Title\n\nFirst words.\n\n- one\n- two\n\n| a | b |\n|---|---|\n| c d | e |\n\n```go\ncode()\n```\n",
		"湖边的桂花开了，用 kite build 一条命令就能重新发布。\n\n> 引用的话\n\n![图](a.jpg)\n",
		"Term\n: Its description\n\nText[^1].\n\n[^1]: A note.\n\n<div>markup</div>\n",
		strings.Repeat("alpha beta ", 60),
		"",
	} {
		doc := render(t, src, nil)
		skim := markdown.Skim(src)
		if skim.Excerpt != doc.Excerpt || skim.WordCount != doc.WordCount || skim.CJKCount != doc.CJKCount {
			t.Errorf("Skim(%q) = %q, %d, %d; Render has %q, %d, %d", src,
				skim.Excerpt, skim.WordCount, skim.CJKCount, doc.Excerpt, doc.WordCount, doc.CJKCount)
		}
	}
}

// A body names the site's own pages and files from the site's root, which is
// right wherever the site is published once the renderer puts the path it is
// published under in front of them.
func TestSiteLinksFollowTheBasePath(t *testing.T) {
	src := "[about](/about/) ![river](/uploads/river.jpg) [again](/blog/about/) " +
		"[cdn](//cdn.example.com/x.js) [out](https://example.com/) [near](notes.md) " +
		"[top](#top) [ref][r]\n\n[r]: /posts/hello/\n"
	doc, err := markdown.New(markdown.Options{BasePath: "/blog/"}).Render(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="/blog/about/"`, `src="/blog/uploads/river.jpg"`, `href="/blog/about/"`,
		`href="//cdn.example.com/x.js"`, `href="https://example.com/"`, `href="notes.md"`,
		`href="#top"`, `href="/blog/posts/hello/"`,
	} {
		if !strings.Contains(doc.HTML, want) {
			t.Errorf("rendered without %s:\n%s", want, doc.HTML)
		}
	}
	if strings.Contains(doc.HTML, "/blog/blog/") {
		t.Errorf("a link already under the path got it twice:\n%s", doc.HTML)
	}

	// At the root of a host a link is written as it is.
	root := render(t, "[about](/about/)", nil)
	if !strings.Contains(root.HTML, `href="/about/"`) {
		t.Errorf("at the root: %s", root.HTML)
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

// A theme labels a code block by its language, so a highlighted block says
// which one it is written in, and a block without one says nothing.
func TestCodeBlocksNameTheirLanguage(t *testing.T) {
	doc := render(t, "```go\nfunc main() {}\n```\n\n```\nplain\n```\n", nil)
	if !strings.Contains(doc.HTML, `<pre class="chroma" data-lang="go"><code>`) {
		t.Errorf("highlighted block does not name its language:\n%s", doc.HTML)
	}
	if strings.Count(doc.HTML, "data-lang") != 1 {
		t.Errorf("a block without a language was given one:\n%s", doc.HTML)
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
