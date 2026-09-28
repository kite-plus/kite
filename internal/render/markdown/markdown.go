// Package markdown renders content bodies to HTML.
//
// There is one pipeline and both runtimes use it: a static build and a live
// server that rendered markdown differently would produce pages that differ
// from the preview the author approved.
package markdown

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Options configures the pipeline.
type Options struct {
	// UnsafeHTML allows raw HTML in markdown through to the output.
	//
	// The default is off. This value changes how every existing document
	// renders, so it is a site-wide decision made once: turning it on later is
	// additive for authors, while turning it off later would silently strip
	// content from pages that already depend on it.
	UnsafeHTML bool

	// Typographer applies smart quotes and dashes.
	Typographer bool

	// HardWraps renders a single newline as a line break.
	HardWraps bool

	// HighlightTheme names the chroma style used for fenced code blocks.
	//
	// It only selects which token classes are emitted, never their colors:
	// highlighting is written as classes so a theme can carry one palette for
	// light and another for dark. Inline colors could not answer a media
	// query, which is why a code block used to stay light on a dark page.
	HighlightTheme string

	// BasePath is the path the site is published under, as "/blog" for a
	// GitHub Pages project site, and empty at the root of its host. A body
	// names the site's own pages and files from the site's root, as
	// "/about/" or "/uploads/river.jpg", which keeps it right wherever the
	// site is published; the renderer puts BasePath in front of them.
	BasePath string
}

// DefaultOptions returns the pipeline defaults.
func DefaultOptions() Options {
	return Options{HighlightTheme: "github"}
}

// Heading is one entry in a document's table of contents.
type Heading struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// ExcerptLimit is about how many characters an excerpt keeps.
const ExcerptLimit = 220

// Document is the result of rendering one body.
type Document struct {
	HTML string
	TOC  []Heading

	// Excerpt is the prose the body opens with, as plain text, cut at a word
	// near ExcerptLimit: what a listing shows of the item.
	Excerpt string

	// Links and Images are what the body links to and shows, in order and as
	// its source writes them, before a site's path is put in front: a theme
	// that shows one of the pictures elsewhere resolves it as it would a
	// cover, and would otherwise add that path twice.
	Links  []string
	Images []string

	// WordCount is how many words a reader reads: those of paragraphs, list
	// items, headings and table cells, but not of code blocks or alt text.
	WordCount int

	// CJKCount is how many of the words are single CJK characters. They are
	// read at a different pace than spaced words, so a reading time needs
	// the two apart.
	CJKCount int

	// Text is the text those words are counted in, a block to a line, for
	// what searches a page rather than shows it.
	Text string
}

// Renderer turns markdown into HTML.
type Renderer struct {
	md goldmark.Markdown
}

// New returns a renderer for the given options.
func New(opts Options) *Renderer {
	if opts.HighlightTheme == "" {
		opts.HighlightTheme = "github"
	}

	extensions := []goldmark.Extender{
		extension.GFM,
		extension.Footnote,
		extension.DefinitionList,
		highlighting.NewHighlighting(
			highlighting.WithStyle(opts.HighlightTheme),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			highlighting.WithCodeBlockOptions(func(c highlighting.CodeBlockContext) []chromahtml.Option {
				lang, ok := c.Language()
				if !ok || len(lang) == 0 {
					return nil
				}
				return []chromahtml.Option{chromahtml.WithPreWrapper(langPre(lang))}
			}),
		),
	}
	if opts.Typographer {
		extensions = append(extensions, extension.Typographer)
	}

	var rendererOpts []renderer.Option
	if opts.UnsafeHTML {
		rendererOpts = append(rendererOpts, html.WithUnsafe())
	}
	if opts.HardWraps {
		rendererOpts = append(rendererOpts, html.WithHardWraps())
	}

	parserOpts := []parser.Option{
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
		parser.WithASTTransformers(util.Prioritized(summaryEnd{}, 100)),
	}
	if base := strings.TrimSuffix(opts.BasePath, "/"); base != "" {
		parserOpts = append(parserOpts, parser.WithASTTransformers(util.Prioritized(siteLinks{base: base}, 1000)))
	}
	rendererOpts = append(rendererOpts, renderer.WithNodeRenderers(util.Prioritized(moreRenderer{}, 100)))

	return &Renderer{md: goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOpts...),
		goldmark.WithRendererOptions(rendererOpts...),
	)}
}

// more stands where an author ended a post's summary with <!--more-->, as
// Hugo and Hexo write it. The excerpt is the prose before it, and the page
// shows nothing in its place.
type more struct{ ast.BaseBlock }

var kindMore = ast.NewNodeKind("More")

func (*more) Kind() ast.NodeKind              { return kindMore }
func (m *more) Dump(source []byte, level int) { ast.DumpHelper(m, source, level, nil, nil) }

// moreRenderer writes nothing for a more node.
type moreRenderer struct{}

func (moreRenderer) RegisterFuncs(r renderer.NodeRendererFuncRegisterer) {
	r.Register(kindMore, func(util.BufWriter, []byte, ast.Node, bool) (ast.WalkStatus, error) {
		return ast.WalkSkipChildren, nil
	})
}

// moreComment is the divider, spaced or not, in any case: <!--more--> or
// <!-- more -->.
var moreComment = regexp.MustCompile(`(?i)^<!--\s*more\s*-->$`)

// summaryEnd turns each divider on a line of its own at the top level of a
// body into a more node.
type summaryEnd struct{}

func (summaryEnd) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	src := reader.Source()
	for n := doc.FirstChild(); n != nil; {
		next := n.NextSibling()
		if isMoreComment(n, src) {
			doc.ReplaceChild(doc, n, &more{})
		}
		n = next
	}
}

func isMoreComment(n ast.Node, src []byte) bool {
	block, ok := n.(*ast.HTMLBlock)
	if !ok || block.HTMLBlockType != ast.HTMLBlockType2 || block.Lines().Len() != 1 || block.HasClosure() {
		return false
	}
	line := block.Lines().At(0)
	return moreComment.Match(bytes.TrimSpace(line.Value(src)))
}

// langPre writes the pre element chroma writes, naming the block's language
// in data-lang so that a theme can label the block.
type langPre string

func (l langPre) Start(code bool, styleAttr string) string {
	if !code {
		return fmt.Sprintf(`<pre%s>`, styleAttr)
	}
	return fmt.Sprintf(`<pre%s data-lang="%s"><code>`, styleAttr, stdhtml.EscapeString(string(l)))
}

func (l langPre) End(code bool) string {
	if !code {
		return `</pre>`
	}
	return `</code></pre>`
}

// siteLinks puts the path a site is published under in front of the links
// and pictures a body names from the site's root. One that already starts
// with the path is left alone, as is anything on another host.
type siteLinks struct{ base string }

func (t siteLinks) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	written := map[ast.Node][]byte{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest *[]byte
		switch node := n.(type) {
		case *ast.Link:
			dest = &node.Destination
		case *ast.Image:
			dest = &node.Destination
		default:
			return ast.WalkContinue, nil
		}
		if rebased, ok := t.rebase(*dest); ok {
			written[n] = *dest
			*dest = rebased
		}
		return ast.WalkContinue, nil
	})
	pc.Set(writtenKey, written)
}

// rebase is dest under the site's path, and whether that changed it.
func (t siteLinks) rebase(dest []byte) ([]byte, bool) {
	d := string(dest)
	if !strings.HasPrefix(d, "/") || strings.HasPrefix(d, "//") || d == t.base || strings.HasPrefix(d, t.base+"/") {
		return dest, false
	}
	return []byte(t.base + d), true
}

// writtenKey holds the destinations siteLinks rebased, by node, as the source
// wrote them.
var writtenKey = parser.NewContextKey()

// asWritten is a node's destination as the source wrote it.
func asWritten(n ast.Node, dest []byte, written map[ast.Node][]byte) string {
	if w, ok := written[n]; ok {
		return string(w)
	}
	return string(dest)
}

// Render converts a markdown body into HTML plus the metadata a theme needs.
func (r *Renderer) Render(source string) (*Document, error) {
	src := []byte(source)
	pc := parser.NewContext()
	root := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(pc))

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, root); err != nil {
		return nil, fmt.Errorf("markdown: render: %w", err)
	}

	doc := &Document{HTML: buf.String()}
	written, _ := pc.Get(writtenKey).(map[ast.Node][]byte)
	if err := collect(root, src, doc, written); err != nil {
		return nil, err
	}
	return doc, nil
}

// collect walks the tree once, gathering everything a theme or a build step
// needs so that nothing has to re-parse the document later.
func collect(root ast.Node, src []byte, doc *Document, written map[ast.Node][]byte) error {
	var plain strings.Builder
	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Heading:
			doc.TOC = append(doc.TOC, Heading{
				Level: node.Level,
				ID:    headingID(node),
				Text:  plainText(node, src),
			})
		case *ast.Link:
			doc.Links = append(doc.Links, asWritten(node, node.Destination, written))
		case *ast.AutoLink:
			doc.Links = append(doc.Links, string(node.URL(src)))
		case *ast.Image:
			doc.Images = append(doc.Images, asWritten(node, node.Destination, written))
		}
		if prose(n) {
			if read := strings.TrimSpace(doc.count(n, src)); read != "" {
				plain.WriteString(read)
				plain.WriteByte('\n')
			}
		}
		return ast.WalkContinue, nil
	})
	doc.Text = strings.TrimSuffix(plain.String(), "\n")
	doc.Excerpt = opening(root, src)
	return err
}

// count adds the words of a prose block to the document's counts, and returns
// the text they were counted in.
func (doc *Document) count(n ast.Node, src []byte) string {
	read := readText(n, src)
	words, cjk := countWords(read)
	doc.WordCount += words
	doc.CJKCount += cjk
	return read
}

// Excerpt is the prose a markdown source opens with, as a Document of it
// would have it, read without rendering the source.
func Excerpt(source string) string {
	src := []byte(source)
	root := excerptParser().Parse(text.NewReader(src), parser.WithContext(parser.NewContext()))
	return opening(root, src)
}

// Skim reads a markdown source without rendering it, for what a list shows of
// it: a Document with only its Excerpt, WordCount, CJKCount and Images, as
// Render would have them.
func Skim(source string) *Document {
	src := []byte(source)
	root := excerptParser().Parse(text.NewReader(src), parser.WithContext(parser.NewContext()))
	doc := &Document{Excerpt: opening(root, src)}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if image, ok := n.(*ast.Image); ok {
			doc.Images = append(doc.Images, string(image.Destination))
		}
		if prose(n) {
			doc.count(n, src)
		}
		return ast.WalkContinue, nil
	})
	return doc
}

// excerptParser reads sources the way the renderer does, so that an excerpt
// sees the same tables, footnotes and lists a page does.
var excerptParser = sync.OnceValue(func() parser.Parser { return New(DefaultOptions()).md.Parser() })

// opening is the prose a document opens with, as plain text: its paragraphs
// and list items in order, without the headings, quotations, tables, code,
// raw HTML and footnotes between them, cut at a word near ExcerptLimit.
//
// An author who ends the summary with <!--more--> has said where it ends, so
// then it is all the prose before the divider, however long, unless there is
// none.
//
// It is read from the tree rather than the source, which is what keeps an
// underscore inside a word and an escaped asterisk: neither is emphasis.
func opening(root ast.Node, src []byte) string {
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if n.Kind() == kindMore {
			if summary := strings.Join(prosePrefix(root, n, src, -1), " "); summary != "" {
				return summary
			}
			break
		}
	}
	return truncate(strings.Join(prosePrefix(root, nil, src, ExcerptLimit), " "), ExcerptLimit)
}

// prosePrefix is the prose of the blocks of root before end, stopping once
// it is longer than limit characters when limit is not negative.
func prosePrefix(root, end ast.Node, src []byte, limit int) []string {
	var parts []string
	length := 0
	for block := root.FirstChild(); block != nil && block != end; block = block.NextSibling() {
		status := ast.WalkContinue
		_ = ast.Walk(block, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch n.(type) {
			case *ast.Heading, *ast.Blockquote, *ast.HTMLBlock, *east.Table, *east.FootnoteList:
				return ast.WalkSkipChildren, nil
			case *ast.Paragraph, *ast.TextBlock:
				if text := strings.Join(strings.Fields(readText(n, src)), " "); text != "" {
					parts = append(parts, text)
					length += utf8.RuneCountInString(text) + 1
				}
				if limit >= 0 && length > limit {
					status = ast.WalkStop
					return ast.WalkStop, nil
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		})
		if status == ast.WalkStop {
			break
		}
	}
	return parts
}

// prose reports a block whose text is read: a paragraph wherever it sits, the
// text of a tight list item, a heading, a table cell or a defined term. A code
// block is skimmed rather than read, and raw HTML is markup, so neither is.
func prose(n ast.Node) bool {
	switch n.(type) {
	case *ast.Paragraph, *ast.TextBlock, *ast.Heading, *east.TableCell, *east.DefinitionTerm:
		return true
	}
	return false
}

func headingID(h *ast.Heading) string {
	raw, ok := h.AttributeString("id")
	if !ok {
		return ""
	}
	id, ok := raw.([]byte)
	if !ok {
		return ""
	}
	return string(id)
}

// plainText concatenates the literal text under a node, which is what a table
// of contents entry and an excerpt need.
func plainText(n ast.Node, src []byte) string { return literal(n, src, false) }

// readText is the text under a node that a reader reads: an image is seen, so
// its alt text is left out of a word count.
func readText(n ast.Node, src []byte) string { return literal(n, src, true) }

func literal(n ast.Node, src []byte, withoutImages bool) string {
	var b strings.Builder
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := child.(type) {
		case *ast.Image:
			if withoutImages {
				return ast.WalkSkipChildren, nil
			}
		case *ast.Text:
			value := t.Segment.Value(src)
			if !t.IsRaw() {
				value = unescape(value)
			}
			b.Write(value)
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			if t.IsCode() {
				// A typographer's quotes and dashes, written as entities.
				b.WriteString(stdhtml.UnescapeString(string(t.Value)))
			} else {
				b.Write(t.Value)
			}
		case *ast.AutoLink:
			b.Write(t.URL(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// unescape is text as the renderer writes it: a backslash escape stands for
// the character it escapes, and an entity or a numeric reference for its
// character. A reference is only resolved between escapes, since an escaped
// ampersand starts none.
func unescape(value []byte) []byte {
	resolve := func(b []byte) []byte { return util.ResolveEntityNames(util.ResolveNumericReferences(b)) }
	var out []byte
	start := 0
	for i := 0; i+1 < len(value); i++ {
		if value[i] == '\\' && util.IsPunct(value[i+1]) {
			out = append(out, resolve(value[start:i])...)
			out = append(out, value[i+1])
			i++
			start = i + 1
		}
	}
	if start == 0 {
		return resolve(value)
	}
	return append(out, resolve(value[start:])...)
}

// countWords counts words, and how many of them are CJK characters.
//
// Splitting on spaces alone counts a whole Chinese paragraph as one word,
// because those scripts put no spaces between words. Each CJK character is
// counted on its own instead, which is also how their writers measure length.
// Punctuation neither starts a word nor ends one, so "don't" stays one word
// and a full-width comma is not counted as one.
func countWords(s string) (words, cjk int) {
	inWord := false
	for _, r := range s {
		switch {
		case unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul):
			cjk++
			inWord = false
		case unicode.IsSpace(r):
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				words++
				inWord = true
			}
		}
	}
	return words + cjk, cjk
}

// truncate cuts s at a word near limit characters and marks the cut, so that
// an excerpt does not end mid-word where it can help it. Text with no spaces
// to cut at, as Chinese has, is cut at the limit.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	cut := string(runes[:limit])
	if i := strings.LastIndexByte(cut, ' '); i > len(cut)/2 {
		cut = cut[:i]
	}
	// Cut inside math between \( and \), or \[ and \], the reader would get
	// its source rather than a formula, so the excerpt stops before it.
	for _, pair := range [][2]string{{`\(`, `\)`}, {`\[`, `\]`}} {
		if open := strings.LastIndex(cut, pair[0]); open >= 0 && open > strings.LastIndex(cut, pair[1]) {
			cut = cut[:open]
		}
	}
	return strings.TrimRight(cut, " ,.;:，、；：") + "…"
}
