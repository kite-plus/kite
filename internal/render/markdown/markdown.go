// Package markdown renders content bodies to HTML.
//
// There is one pipeline and both runtimes use it: a static build and a live
// server that rendered markdown differently would produce pages that differ
// from the preview the author approved.
package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
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
	HighlightTheme string
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

// Document is the result of rendering one body.
type Document struct {
	HTML      string
	TOC       []Heading
	Excerpt   string
	Links     []string
	Images    []string
	WordCount int
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
		highlighting.NewHighlighting(highlighting.WithStyle(opts.HighlightTheme)),
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

	return &Renderer{md: goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(rendererOpts...),
	)}
}

// Render converts a markdown body into HTML plus the metadata a theme needs.
func (r *Renderer) Render(source string) (*Document, error) {
	src := []byte(source)
	root := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(parser.NewContext()))

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, root); err != nil {
		return nil, fmt.Errorf("markdown: render: %w", err)
	}

	doc := &Document{HTML: buf.String()}
	if err := collect(root, src, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// collect walks the tree once, gathering everything a theme or a build step
// needs so that nothing has to re-parse the document later.
func collect(root ast.Node, src []byte, doc *Document) error {
	return ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
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
			doc.Links = append(doc.Links, string(node.Destination))
		case *ast.AutoLink:
			doc.Links = append(doc.Links, string(node.URL(src)))
		case *ast.Image:
			doc.Images = append(doc.Images, string(node.Destination))
		case *ast.Paragraph:
			text := strings.TrimSpace(plainText(node, src))
			doc.WordCount += len(strings.Fields(text))
			if doc.Excerpt == "" && text != "" {
				doc.Excerpt = truncate(text, 200)
			}
		}
		return ast.WalkContinue, nil
	})
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
// of contents entry and a word count need.
func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := child.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		case *ast.AutoLink:
			b.Write(t.URL(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}
