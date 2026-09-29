package markdown_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/render/markdown"
)

// framing draws a body's pictures in a frame of its own, but for those whose
// address says to leave them.
type framing struct{ drawing }

func (framing) Picture(p markdown.Picture) (string, bool, error) {
	switch {
	case strings.HasPrefix(p.Destination, "plain"):
		return "", false, nil
	case p.Destination == "broken.png":
		return "", false, errors.New("the template failed")
	}
	return fmt.Sprintf(`<figure data-dest="%s" data-src="%s" title="%s">%s</figure>`,
		p.Destination, p.Src, p.Title, p.Text), true, nil
}

// A site or its theme draws the pictures a body shows, as Hugo's render
// hooks do: handed each as the body writes it and as a page links it, with
// its words and title. One it leaves is drawn as markdown draws it.
func TestAPictureIsDrawnByTheTemplateForIt(t *testing.T) {
	r := markdown.New(markdown.Options{BasePath: "/blog"})
	doc, err := r.Render("![A *river* bend](river.jpg \"Upstream\")\n\n![Root](/uploads/a.png)\n\n![Left](plain.png)\n", framing{})
	if err != nil {
		t.Fatal(err)
	}
	contains(t, doc.HTML,
		`<p><figure data-dest="river.jpg" data-src="river.jpg" title="Upstream">A river bend</figure></p>`,
		`<figure data-dest="/uploads/a.png" data-src="/blog/uploads/a.png" title="">Root</figure>`,
		`<p><img src="plain.png" alt="Left"></p>`,
	)
	if got := strings.Join(doc.Images, " "); got != "river.jpg /uploads/a.png plain.png" {
		t.Errorf("Images = %q, want the pictures as the body writes them", got)
	}

	if _, err := r.Render("![x](broken.png)\n", framing{}); err == nil || !strings.Contains(err.Error(), "the template failed") {
		t.Errorf("a failing template: %v", err)
	}
	plain, err := r.Render("![x](river.jpg)\n", drawing{})
	if err != nil || !strings.Contains(plain.HTML, `<img src="river.jpg" alt="x">`) {
		t.Errorf("without a template: %v %s", err, plain.HTML)
	}
}
