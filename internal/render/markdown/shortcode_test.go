package markdown_test

import (
	"errors"
	"fmt"
	"html"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/render/markdown"
)

// drawing draws a call as an element that names it and what it was handed,
// which is what a test reads a template's view of the call off.
type drawing struct{}

func (drawing) Draw(c *markdown.Call) (string, error) {
	switch c.Name {
	case "private":
		return "", nil // keeps what it encloses off the page
	case "broken":
		return "", errors.New("the template failed")
	case "ref":
		return "/posts/" + fmt.Sprint(c.Positional[0]) + "/?a=1&amp;b=2", nil
	}
	inner, err := c.Inner()
	if err != nil {
		return "", err
	}
	var params []string
	for _, v := range c.Positional {
		params = append(params, fmt.Sprintf("%T:%v", v, v))
	}
	for _, k := range slices.Sorted(maps.Keys(c.Named)) {
		params = append(params, fmt.Sprintf("%s=%T:%v", k, c.Named[k], c.Named[k]))
	}
	parent := ""
	if c.Parent != nil {
		parent = c.Parent.Name
	}
	return fmt.Sprintf(`<sc name="%s" params="%s" ordinal="%d" parent="%s">%s</sc>`,
		c.Name, html.EscapeString(strings.Join(params, " ")), c.Ordinal, parent, inner), nil
}

func draw(t *testing.T, src string, mutate func(*markdown.Options)) *markdown.Document {
	t.Helper()
	opts := markdown.DefaultOptions()
	if mutate != nil {
		mutate(&opts)
	}
	doc, err := markdown.New(opts).Render(src, drawing{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return doc
}

func contains(t *testing.T, html string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("output missing %q:\n%s", want, html)
		}
	}
}

// A tag on a line of its own stands for a block, as a figure or a video is,
// so no paragraph wraps what it draws; one inside a line is a word of it.
func TestAShortcodeOnALineOfItsOwnIsABlock(t *testing.T) {
	doc := draw(t, `Before.

{{< figure src="a.jpg" alt="A \"quoted\" word" >}}

Press {{< kbd Ctrl >}} to go on.
`, nil)
	contains(t, doc.HTML,
		"<p>Before.</p>\n"+`<sc name="figure" params="alt=string:A &#34;quoted&#34; word src=string:a.jpg"`,
		`<p>Press <sc name="kbd" params="string:Ctrl" ordinal="1" parent=""></sc> to go on.</p>`,
	)
	if strings.Contains(doc.HTML, `<p><sc name="figure"`) {
		t.Errorf("a paragraph wraps a block:\n%s", doc.HTML)
	}
}

// A pair encloses markdown of its own, which the template is handed rendered:
// the blocks between two tags of their own lines, or the words between two
// tags of a line.
func TestAPairEnclosesMarkdown(t *testing.T) {
	doc := draw(t, `{{< note >}}
Some **bold** words.

- one
- two
{{< /note >}}

A {{< mark >}}marked *word*{{< /mark >}} here.
`, nil)
	contains(t, doc.HTML,
		`<sc name="note" params="" ordinal="0" parent=""><p>Some <strong>bold</strong> words.</p>`+"\n<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n</sc>",
		`<p>A <sc name="mark" params="" ordinal="1" parent="">marked <em>word</em></sc> here.</p>`,
	)
}

// Hugo sites write both kinds of delimiter; they are read alike.
func TestPercentDelimitersAreReadAlike(t *testing.T) {
	doc := draw(t, "{{% note %}}\nInside.\n{{% /note %}}\n", nil)
	contains(t, doc.HTML, `<sc name="note" params="" ordinal="0" parent=""><p>Inside.</p>`)
}

// Each call knows the call it sits in and how many came before it there, so
// a gallery can number its pictures.
func TestNestedCallsKnowTheirParentAndOrdinal(t *testing.T) {
	doc := draw(t, `{{< gallery >}}
{{< pic "a.jpg" >}}
{{< pic "b.jpg" >}}
{{< /gallery >}}

{{< pic "c.jpg" >}}
`, nil)
	contains(t, doc.HTML,
		`<sc name="pic" params="string:a.jpg" ordinal="0" parent="gallery">`,
		`<sc name="pic" params="string:b.jpg" ordinal="1" parent="gallery">`,
		`<sc name="pic" params="string:c.jpg" ordinal="1" parent="">`,
	)
}

// A word written without quotes is the value it reads as, as in Hugo; a quoted
// one is always text, and a value in backquotes is taken as it is.
func TestParametersKeepTheirTypes(t *testing.T) {
	doc := draw(t, "{{< p \"7\" 7 -2 1.5 true word `raw \\\"x\\\"` >}}\n\n{{< n on=false count=3 title=\"Hi there\" >}}\n", nil)
	contains(t, doc.HTML,
		`params="string:7 int:7 int:-2 float64:1.5 bool:true string:word string:raw \&#34;x\&#34;"`,
		`params="count=int:3 on=bool:false title=string:Hi there"`,
	)
}

// A tag may run over several lines, as a figure with many parameters is often
// written.
func TestATagMaySpanLines(t *testing.T) {
	doc := draw(t, "{{< figure\n  src=\"a.jpg\"\n  caption=\"A picture\"\n>}}\n\nAfter.\n", nil)
	contains(t, doc.HTML, `<sc name="figure" params="caption=string:A picture src=string:a.jpg"`, "<p>After.</p>")
}

// Code shows a tag as written, and a tag commented out as Hugo does it shows
// the tag it stands for, in code and out of it, so a post about shortcodes
// can be written.
func TestCodeAndCommentedTagsShowTheTag(t *testing.T) {
	doc := draw(t, "Use `{{< figure >}}` inline.\n\n```\n{{< figure >}}\n{{</* figure src=\"x\" */>}}\n```\n\nOr write {{</* figure */>}} in text.\n", nil)
	contains(t, doc.HTML,
		"<code>{{&lt; figure &gt;}}</code>",
		"{{&lt; figure &gt;}}\n{{&lt; figure src=&quot;x&quot; &gt;}}",
		"<p>Or write {{&lt; figure &gt;}} in text.</p>",
	)
	if strings.Contains(doc.HTML, "<sc") {
		t.Errorf("a tag in code or commented out was drawn:\n%s", doc.HTML)
	}
}

// Hugo sites link to other posts through a shortcode in the address.
func TestATagMayStandInALinksAddress(t *testing.T) {
	doc := draw(t, "See [the other post]({{< ref \"other\" >}}).\n", func(o *markdown.Options) { o.BasePath = "/blog" })
	contains(t, doc.HTML, `<a href="/blog/posts/other/?a=1&amp;b=2">the other post</a>`)
}

// Raw HTML is passed through as it is, with the tags in it drawn.
func TestTagsInRawHTMLAreDrawn(t *testing.T) {
	doc := draw(t, "<div class=\"grid\">\n{{< pic \"a.jpg\" >}}\n{{< box >}}<b>in</b>{{< /box >}}\n</div>\n", func(o *markdown.Options) { o.UnsafeHTML = true })
	contains(t, doc.HTML,
		`<div class="grid">`+"\n"+`<sc name="pic" params="string:a.jpg" ordinal="0" parent=""></sc>`,
		`<sc name="box" params="" ordinal="1" parent=""><b>in</b></sc>`,
	)
}

// A tag of its own line in the middle of a paragraph stays one of its words,
// as CommonMark keeps an HTML tag there; a list item may hold one as a block.
func TestATagInsideAParagraphStaysInIt(t *testing.T) {
	doc := draw(t, "Line one\n{{< icon star >}}\nline two.\n\n- {{< pic \"a.jpg\" >}}\n- text\n", nil)
	contains(t, doc.HTML,
		"<p>Line one\n"+`<sc name="icon" params="string:star" ordinal="0" parent=""></sc>`+"\nline two.</p>",
		"<li>\n"+`<sc name="pic"`,
	)
}

// Closing a block at the end of the last paragraph it encloses is a way of
// writing a pair too.
func TestABlockMayCloseAtTheEndOfItsLastParagraph(t *testing.T) {
	doc := draw(t, "{{< note >}}\nText. {{< /note >}}\n\nAfter.\n", nil)
	contains(t, doc.HTML, `<sc name="note" params="" ordinal="0" parent=""><p>Text.</p>`, "<p>After.</p>")
}

// What a shortcode keeps off the page is not the page's words, text or
// contents; what one shows is.
func TestOnlyShownInnerContentCounts(t *testing.T) {
	doc := draw(t, `Visible words here.

{{< private >}}
## Secret heading

secret words that stay hidden
{{< /private >}}

{{< note >}}
## Shown heading

three shown words
{{< /note >}}
`, nil)
	if strings.Contains(doc.HTML, "secret") || strings.Contains(doc.Text, "secret") {
		t.Errorf("hidden content reached the page:\n%s\n%s", doc.HTML, doc.Text)
	}
	if doc.WordCount != 8 {
		t.Errorf("WordCount = %d, want 8 (3 visible, 2 of the shown heading, 3 shown)", doc.WordCount)
	}
	var toc []string
	for _, h := range doc.TOC {
		toc = append(toc, h.Text)
	}
	if !slices.Equal(toc, []string{"Shown heading"}) {
		t.Errorf("TOC = %q, want only the shown heading", toc)
	}
}

// An excerpt is the opening prose, which a shortcode of its own line is not,
// read or rendered; the words a pair of a line encloses are.
func TestAnExcerptLeavesBlockShortcodesOut(t *testing.T) {
	const src = "{{< figure src=\"a.jpg\" >}}\n\n{{< note >}}\nA note.\n{{< /note >}}\n\nThe {{< mark >}}opening{{< /mark >}} words.\n"
	if got := draw(t, src, nil).Excerpt; got != "The opening words." {
		t.Errorf("rendered excerpt = %q", got)
	}
	skim := markdown.Skim(src)
	if skim.Excerpt != "The opening words." {
		t.Errorf("skimmed excerpt = %q", skim.Excerpt)
	}
	if skim.WordCount != 5 {
		t.Errorf("skimmed WordCount = %d, want 5", skim.WordCount)
	}
	if got := markdown.Excerpt(src); got != "The opening words." {
		t.Errorf("Excerpt = %q", got)
	}
}

// A problem is reported with the line it is on, and a shortcode nothing
// draws is a problem rather than text printed as it was written.
func TestProblemsNameTheirLine(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
	}{
		{"undefined", "Text.\n\n{{< gallery >}}\n", "line 3: shortcode gallery: not defined"},
		{"template", "{{< broken >}}\n", "line 1: shortcode broken: the template failed"},
		{"nested", "{{< note >}}\nText.\n\n{{< broken >}}\n{{< /note >}}\n", "line 4: shortcode broken: the template failed"},
		{"unclosed", "Text {{< figure src=\"a.jpg\"\n", "line 1: shortcode figure: the tag is never closed with >}}"},
		{"quote", "{{< figure src=\"a.jpg >}}\n", "line 1: shortcode figure: a quoted value is never closed"},
		{"mixed", "{{< figure \"a.jpg\" alt=x >}}\n", "line 1: shortcode figure: parameters are given by name or in order, not both"},
		{"orphan", "Text.\n{{< /note >}}\n", "line 2: shortcode note: the closing tag has no opening tag before it"},
		{"name", "{{< >}}\n", "line 1: shortcode: a tag needs a name"},
		{"levels", "- {{< note >}}\n  Text.\n\n{{< /note >}}\n", "line 1: shortcode note: its closing tag on line 4 is not at the level of its opening tag"},
		{"address", "[x]({{< note >}}{{< /note >}})\n", "line 1: shortcode note: a shortcode with a closing tag cannot stand in a link's address"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var sc markdown.Shortcodes = drawing{}
			if c.name == "undefined" {
				sc = nil
			}
			_, err := markdown.New(markdown.DefaultOptions()).Render(c.src, sc)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want one containing %q", err, c.want)
			}
			var se *markdown.ShortcodeError
			if !errors.As(err, &se) {
				t.Errorf("%T is not a ShortcodeError", err)
			}
		})
	}
}

// A body without tags renders exactly as it did before shortcodes existed.
func TestABodyWithoutTagsIsUntouched(t *testing.T) {
	const src = "# Title\n\nA {brace} and {{ .Go }} template, <b>bold</b>.\n"
	with, err := markdown.New(markdown.DefaultOptions()).Render(src, drawing{})
	if err != nil {
		t.Fatal(err)
	}
	without := render(t, src, nil)
	if with.HTML != without.HTML {
		t.Errorf("drawing changed a body without tags:\n%s\n%s", with.HTML, without.HTML)
	}
}

// A heading that ends with a shortcode is reached by its words, without the
// space before the call, and a tag a backslash kept from being read is drawn
// on the page but leaves nothing of itself in the page's text.
func TestWhatIsAroundATagIsReadWithoutIt(t *testing.T) {
	doc := draw(t, "## Stars {{< icon star >}}\n\nA \\{{< icon x >}} b.\n", nil)
	if len(doc.TOC) != 1 || doc.TOC[0].ID != "stars" || doc.TOC[0].Text != "Stars" {
		t.Errorf("TOC = %+v, want the heading Stars at #stars", doc.TOC)
	}
	contains(t, doc.HTML, `<h2 id="stars">Stars <sc name="icon"`, `<p>A <sc name="icon" params="string:x"`)
	if doc.Excerpt != "A b." || strings.Contains(doc.Text, "") {
		t.Errorf("excerpt %q and text %q keep a token", doc.Excerpt, doc.Text)
	}
}
