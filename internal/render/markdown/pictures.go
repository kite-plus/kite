package markdown

import (
	"sync"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// Pictures draws the pictures a body shows, for a site or a theme that has a
// template for them, as Hugo's render hooks do. The Shortcodes a body is
// rendered with may draw them too.
type Pictures interface {
	// Picture returns the HTML a picture is shown with, and false to leave
	// it as markdown writes it.
	Picture(p Picture) (string, bool, error)
}

// Picture is a picture a body shows: ![Text](Destination "Title").
type Picture struct {
	// Destination is the picture's address as the body writes it, and Src
	// the address a page shows it from without a template: under the site's
	// path when the body names it from the site's root.
	Destination string
	Src         string

	// Text is what the picture says in words, for those who cannot see it.
	Text  string
	Title string
}

// drawing holds what draws the pictures of each body being rendered, by the
// root of its tree, since a node renderer is shared by every render.
var drawing sync.Map

type picturing struct {
	pictures Pictures
	written  map[ast.Node][]byte
}

// pictureRenderer draws a picture with the Pictures its body is rendered
// with, and as markdown writes it when they leave it.
type pictureRenderer struct{ plain renderer.NodeRendererFunc }

// newPictureRenderer takes the plain way of drawing a picture from goldmark,
// with the options the site renders HTML with.
func newPictureRenderer(opts ...html.Option) pictureRenderer {
	var plain capture
	html.NewRenderer(opts...).RegisterFuncs(&plain)
	return pictureRenderer{plain: plain.image}
}

type capture struct{ image renderer.NodeRendererFunc }

func (c *capture) Register(kind ast.NodeKind, f renderer.NodeRendererFunc) {
	if kind == ast.KindImage {
		c.image = f
	}
}

func (r pictureRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.render)
}

func (r pictureRenderer) render(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		if p := picturingOf(n); p != nil {
			image := n.(*ast.Image)
			out, drawn, err := p.pictures.Picture(Picture{
				Destination: asWritten(image, image.Destination, p.written),
				Src:         string(image.Destination),
				Text:        plainText(image, source),
				Title:       string(image.Title),
			})
			if err != nil {
				return ast.WalkStop, err
			}
			if drawn {
				_, _ = w.WriteString(out)
				return ast.WalkSkipChildren, nil
			}
		}
	}
	return r.plain(w, source, n, entering)
}

func picturingOf(n ast.Node) *picturing {
	for n.Parent() != nil {
		n = n.Parent()
	}
	p, _ := drawing.Load(n)
	pic, _ := p.(*picturing)
	return pic
}
