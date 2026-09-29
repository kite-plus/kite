package markdown

import (
	"bytes"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Shortcodes are Hugo's, so that content moved from a Hugo site keeps working:
// a body calls a template by name, as {{< figure src="a.jpg" >}}, or around
// markdown of its own, as {{< note >}}Read this.{{< /note >}}, and the site
// or its theme provides the template. {{% %}} is read the same way.
//
// A tag on a line of its own is a block, which no paragraph wraps; a pair of
// them encloses the blocks between. A tag inside a line is inline, and a pair
// of those encloses the words between. A tag in code is shown as written, and
// {{</* name */>}} shows the tag it comments out anywhere.

// Shortcodes draws the shortcodes a body calls.
type Shortcodes interface {
	// Draw returns the HTML a call stands for.
	Draw(call *Call) (string, error)
}

// Call is one shortcode a body calls.
type Call struct {
	// Name is what the body calls it, as "figure" or "docs/note".
	Name string

	// Positional holds the parameters a call gives in order, and Named those
	// it gives by name; a call gives them one way or the other. A quoted
	// value is text, and a word that reads as true, false or a number is
	// that value.
	Positional []any
	Named      map[string]any

	// Line is the line of the body the call starts on, from 1.
	Line int

	// Ordinal counts the calls before this one with the same parent, from 0.
	Ordinal int

	// Parent is the call whose inner content holds this one, nil at the top
	// of the body.
	Parent *Call

	// Paired says the call has a closing tag, and RawInner is what the two
	// tags enclose, as the body writes it.
	Paired   bool
	RawInner string

	inner    func() (string, error)
	rendered *string
	shown    bool
}

// Inner is what the call encloses, rendered as markdown; empty for a call
// without a closing tag.
//
// Only the words of an inner content a template asks for count as the page's
// words, so a shortcode that keeps its content off the page keeps it out of
// the page's word count and text as well.
func (c *Call) Inner() (string, error) {
	if c.inner == nil {
		return "", nil
	}
	c.shown = true
	if c.rendered == nil {
		html, err := c.inner()
		if err != nil {
			return "", err
		}
		c.rendered = &html
	}
	return *c.rendered, nil
}

// ShortcodeError is a shortcode a body calls that cannot be drawn, or a tag
// that cannot be read, with the line of the body it is on.
type ShortcodeError struct {
	Line int
	Name string
	Err  error
}

func (e *ShortcodeError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("line %d: shortcode: %v", e.Line, e.Err)
	}
	return fmt.Sprintf("line %d: shortcode %s: %v", e.Line, e.Name, e.Err)
}

func (e *ShortcodeError) Unwrap() error { return e.Err }

// CallsShortcodes reports whether a body may call a shortcode, cheaply: a tag
// in code counts too.
func CallsShortcodes(source string) bool {
	return strings.Contains(source, "{{<") || strings.Contains(source, "{{%")
}

// A token stands in for a tag while the markdown is parsed: an opening brace,
// which is what lets an inline parser see it, and the tag's number between two
// characters of Unicode's private use area, which text does not hold. It has
// no space, so it can stand in a link's address as well.
const (
	tokenStart = "{"
	tokenEnd   = ""
)

func tokenOf(n int) string { return tokenStart + strconv.Itoa(n) + tokenEnd }

// tokenAt reads the token at the start of b, and how long it is.
func tokenAt[T ~string | ~[]byte](b T) (n, width int, ok bool) {
	if len(b) < len(tokenStart) || string(b[:len(tokenStart)]) != tokenStart {
		return 0, 0, false
	}
	i := len(tokenStart)
	j := i
	for j < len(b) && '0' <= b[j] && b[j] <= '9' {
		j++
	}
	if j == i || len(b)-j < len(tokenEnd) || string(b[j:j+len(tokenEnd)]) != tokenEnd {
		return 0, 0, false
	}
	n, err := strconv.Atoi(string(b[i:j]))
	if err != nil {
		return 0, 0, false
	}
	return n, j + len(tokenEnd), true
}

type role uint8

const (
	roleSingle role = iota
	roleOpen
	roleClose
)

// token is what a token stands for: one tag of a call.
type token struct {
	call *Call
	role role
	line int
	text string // the tag as written

	// ends says nothing but space follows the tag on its line.
	ends bool
}

// session is one reading of a body that calls shortcodes.
type session struct {
	calls  []*Call
	tokens []token
	opens  map[*Call]int // the token of a pair's opening tag
	closes map[*Call]int // and of its closing one

	// draw draws the calls; nil when a body is read rather than rendered.
	draw Shortcodes
	// render renders a node of the tree the body was parsed into.
	render func(io.Writer, ast.Node) error

	// blocks holds the pairs whose opening tag is a block.
	blocks map[*Call]bool

	err error
}

var sessionKey = parser.NewContextKey()

func sessionOf(pc parser.Context) *session {
	s, _ := pc.Get(sessionKey).(*session)
	return s
}

func (s *session) fail(c *Call, format string, args ...any) {
	if s.err == nil {
		s.err = &ShortcodeError{Line: c.Line, Name: c.Name, Err: fmt.Errorf(format, args...)}
	}
}

// prepare finds the shortcode tags of a body and returns the body with a
// token in place of each, and the calls they make. A body that calls none
// comes back as it is, with no session.
//
// Tags in code are left as they are. Which parts of a body are code is what
// the parser says, so the body is parsed once as it is written to find out.
func prepare(p parser.Parser, source string) (string, *session, error) {
	if !CallsShortcodes(source) {
		return source, nil, nil
	}
	code := codeRegions(p, source)
	inCode := func(at int) bool {
		i := sort.Search(len(code), func(i int) bool { return code[i][1] > at })
		return i < len(code) && code[i][0] <= at
	}

	type edit struct {
		start, end int
		text       string // what replaces the span; a tag's token when tag >= 0
		tag        int
	}
	var tags []tag
	var edits []edit
	line, counted := 1, 0
	for i := 0; ; {
		j := indexTag(source, i)
		if j < 0 {
			break
		}
		line += strings.Count(source[counted:j], "\n")
		counted = j
		if literal, end, ok := escaped(source, j); ok {
			if !inCode(j) {
				literal = literally(literal)
			}
			edits = append(edits, edit{start: j, end: end, text: literal, tag: -1})
			i = end
			continue
		}
		if inCode(j) {
			i = j + 3
			continue
		}
		t, err := lexTag(source, j, line)
		if err != nil {
			return "", nil, err
		}
		edits = append(edits, edit{start: j, end: t.end, tag: len(tags)})
		tags = append(tags, t)
		i = t.end
	}
	if len(tags) == 0 && len(edits) == 0 {
		return source, nil, nil
	}

	s, err := pair(tags, source)
	if err != nil {
		return "", nil, err
	}

	var b strings.Builder
	at := make([]int, len(tags)) // where each token lands
	last := 0
	for _, e := range edits {
		b.WriteString(source[last:e.start])
		if e.tag >= 0 {
			at[e.tag] = b.Len()
			b.WriteString(tokenOf(e.tag))
		} else {
			b.WriteString(e.text)
		}
		last = e.end
	}
	b.WriteString(source[last:])
	body := b.String()
	for n := range s.tokens {
		rest := body[at[n]+len(tokenOf(n)):]
		if stop := strings.IndexByte(rest, '\n'); stop >= 0 {
			rest = rest[:stop]
		}
		s.tokens[n].ends = strings.TrimSpace(rest) == ""
	}
	if len(tags) == 0 {
		return body, nil, nil
	}
	return body, s, nil
}

// indexTag finds the next {{< or {{% at or after i.
func indexTag(s string, i int) int {
	for {
		j := strings.Index(s[i:], "{{")
		if j < 0 {
			return -1
		}
		j += i
		if j+2 < len(s) && (s[j+2] == '<' || s[j+2] == '%') {
			return j
		}
		i = j + 1
	}
}

// codeRegions lists the spans of a body that are code, in order: the lines of
// code blocks and the text of code spans.
func codeRegions(p parser.Parser, source string) [][2]int {
	src := []byte(source)
	var out [][2]int
	_ = ast.Walk(p.Parse(text.NewReader(src)), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			if lines := n.Lines(); lines.Len() > 0 {
				out = append(out, [2]int{lines.At(0).Start, lines.At(lines.Len() - 1).Stop})
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan:
			first, last := n.FirstChild(), n.LastChild()
			if a, ok := first.(*ast.Text); ok {
				if z, ok := last.(*ast.Text); ok {
					out = append(out, [2]int{a.Segment.Start, z.Segment.Stop})
				}
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return out
}

// literally writes text for markdown to show as it is, with every ASCII
// punctuation character as a numeric character reference, which markdown and
// HTML both read back as the character.
func literally(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if c := s[i]; util.IsPunct(c) {
			b.WriteString("&#" + strconv.Itoa(int(c)) + ";")
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// pair matches each closing tag with the nearest opening tag of the same name
// before it. A tag opened inside a pair and not closed before the pair is has
// no closing tag, like one that closes itself. Nesting then gives each call
// its parent and its ordinal.
func pair(tags []tag, source string) (*session, error) {
	s := &session{
		tokens: make([]token, len(tags)),
		opens:  make(map[*Call]int),
		closes: make(map[*Call]int),
		blocks: make(map[*Call]bool),
	}
	var open []int
	for i, t := range tags {
		s.tokens[i] = token{line: t.line, text: source[t.start:t.end]}
		if !t.closing {
			c := &Call{Name: t.name, Positional: t.positional, Named: t.named, Line: t.line}
			s.calls = append(s.calls, c)
			s.tokens[i].call = c
			if !t.selfClosed {
				open = append(open, i)
			}
			continue
		}
		k := len(open) - 1
		for k >= 0 && tags[open[k]].name != t.name {
			k--
		}
		if k < 0 {
			return nil, &ShortcodeError{Line: t.line, Name: t.name,
				Err: fmt.Errorf("the closing tag has no opening tag before it")}
		}
		o := open[k]
		open = open[:k]
		c := s.tokens[o].call
		c.Paired, c.RawInner = true, source[tags[o].end:t.start]
		s.tokens[o].role = roleOpen
		s.tokens[i].call, s.tokens[i].role = c, roleClose
		s.opens[c], s.closes[c] = o, i
	}

	var within []*Call
	count := make(map[*Call]int)
	for _, tok := range s.tokens {
		if tok.role == roleClose {
			within = within[:len(within)-1]
			continue
		}
		c := tok.call
		if len(within) > 0 {
			c.Parent = within[len(within)-1]
		}
		c.Ordinal = count[c.Parent]
		count[c.Parent]++
		if tok.role == roleOpen {
			within = append(within, c)
		}
	}
	return s, nil
}

// tagNode is a tag in the tree, block or inline. An opening tag, once its
// closing tag is found, holds what the two enclose.
type tagNode interface {
	ast.Node
	number() int
	session() *session
	hide(bool)
	hidden() bool
}

type tagState struct {
	n      int
	s      *session
	hiding bool
}

func (t *tagState) number() int       { return t.n }
func (t *tagState) session() *session { return t.s }
func (t *tagState) hide(h bool)       { t.hiding = h }
func (t *tagState) hidden() bool      { return t.hiding }

// tagBlock is a tag on a line of its own.
type tagBlock struct {
	ast.BaseBlock
	tagState
}

// tagInline is a tag inside a line.
type tagInline struct {
	ast.BaseInline
	tagState
}

var (
	kindTagBlock  = ast.NewNodeKind("ShortcodeBlock")
	kindTagInline = ast.NewNodeKind("ShortcodeInline")
)

func (*tagBlock) Kind() ast.NodeKind  { return kindTagBlock }
func (*tagInline) Kind() ast.NodeKind { return kindTagInline }

func (t *tagBlock) Dump(source []byte, level int) {
	ast.DumpHelper(t, source, level, map[string]string{"Token": strconv.Itoa(t.n)}, nil)
}

func (t *tagInline) Dump(source []byte, level int) {
	ast.DumpHelper(t, source, level, map[string]string{"Token": strconv.Itoa(t.n)}, nil)
}

// tagLines reads a line that holds nothing but a tag as a block. Pairs may
// interrupt a paragraph, since a paragraph ends where the blocks a pair
// encloses start or stop; a tag that closes itself may not, and stays a word
// of the paragraph instead, as CommonMark keeps an HTML tag of its own line.
//
// An opening tag is a block when its closing tag ends a line as well: on a
// line of its own, or after the last words of what the pair encloses. One
// closed in the middle of a line opens a pair of words.
type tagLines struct{ pairs bool }

func (tagLines) Trigger() []byte { return []byte{'{'} }

func (p tagLines) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	s := sessionOf(pc)
	pos := pc.BlockOffset()
	if s == nil || pos < 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()
	n, width, ok := tokenAt(line[pos:])
	if !ok || n >= len(s.tokens) || !util.IsBlank(line[pos+width:]) {
		return nil, parser.NoChildren
	}
	tok := s.tokens[n]
	switch {
	case !p.pairs && tok.role == roleSingle:
	case p.pairs && tok.role == roleOpen && s.tokens[s.closes[tok.call]].ends:
		s.blocks[tok.call] = true
	case p.pairs && tok.role == roleClose && s.blocks[tok.call]:
	default:
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &tagBlock{tagState: tagState{n: n, s: s}}, parser.NoChildren
}

func (tagLines) Continue(ast.Node, text.Reader, parser.Context) parser.State { return parser.Close }
func (tagLines) Close(ast.Node, text.Reader, parser.Context)                 {}
func (p tagLines) CanInterruptParagraph() bool                               { return p.pairs }
func (tagLines) CanAcceptIndentedLine() bool                                 { return false }

// tagWords reads a tag inside a line.
type tagWords struct{}

func (tagWords) Trigger() []byte { return []byte{'{'} }

func (tagWords) Parse(_ ast.Node, block text.Reader, pc parser.Context) ast.Node {
	s := sessionOf(pc)
	if s == nil {
		return nil
	}
	line, _ := block.PeekLine()
	n, width, ok := tokenAt(line)
	if !ok || n >= len(s.tokens) {
		return nil
	}
	block.Advance(width)
	return &tagInline{tagState: tagState{n: n, s: s}}
}

// pairTags moves what each pair of tags encloses into its opening tag. A pair
// is found in the tree only if both its tags are at one level: two blocks
// among the same blocks, or two words of the same paragraph.
type pairTags struct{}

func (pairTags) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	s := sessionOf(pc)
	if s == nil {
		return
	}
	nodes := make(map[int]ast.Node)
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(tagNode); ok && entering {
			nodes[t.number()] = n
		}
		return ast.WalkContinue, nil
	})
	for _, c := range s.calls {
		if !c.Paired {
			continue
		}
		open, shut := nodes[s.opens[c]], nodes[s.closes[c]]
		switch {
		case open == nil && shut == nil:
			continue // both in raw HTML, where they are paired once it is written
		case open == nil || shut == nil:
			s.fail(c, "its tags on lines %d and %d must both be in markdown, or both in one piece of HTML",
				c.Line, s.tokens[s.closes[c]].line)
			return
		}
		if !enclose(open, shut, reader.Source()) {
			s.fail(c, "its closing tag on line %d is not at the level of its opening tag: "+
				"put both on lines of their own, or both in one paragraph", s.tokens[s.closes[c]].line)
			return
		}
	}
}

// enclose moves everything between an opening tag and its closing tag into
// the opening tag, and drops the closing one.
//
// A block that opens a pair may also be closed at the end of the last of the
// blocks it encloses, as {{< note >}} on a line and "Text. {{< /note >}}" on
// the next are.
func enclose(open, shut ast.Node, src []byte) bool {
	parent := open.Parent()
	last := shut
	if shut.Parent() != parent {
		if _, ok := open.(*tagBlock); !ok {
			return false
		}
		for last != nil && last.Parent() != parent {
			for after := last.NextSibling(); after != nil; after = after.NextSibling() {
				if !blank(after, src) {
					return false
				}
			}
			last = last.Parent()
		}
		if last == nil || !follows(last, open) {
			return false
		}
		if words, ok := shut.PreviousSibling().(*ast.Text); ok {
			words.Segment = words.Segment.TrimRightSpace(src)
		}
		shut.Parent().RemoveChild(shut.Parent(), shut)
	} else if !follows(shut, open) {
		return false
	}

	for n := open.NextSibling(); n != nil; {
		next := n.NextSibling()
		if n == shut {
			parent.RemoveChild(parent, n)
			break
		}
		open.AppendChild(open, n)
		if n == last {
			break
		}
		n = next
	}
	return true
}

// follows reports whether n is a later sibling of first.
func follows(n, first ast.Node) bool {
	for x := first.NextSibling(); x != nil; x = x.NextSibling() {
		if x == n {
			return true
		}
	}
	return false
}

// blank reports whether a node holds nothing but space.
func blank(n ast.Node, src []byte) bool {
	t, ok := n.(*ast.Text)
	return ok && util.IsBlank(t.Segment.Value(src))
}

// tagAddresses draws the tags in the addresses and titles of links and
// pictures, as [the other post]({{< ref "other.md" >}}) writes one, into the
// text they stand for. A body that is read rather than rendered gets its tags
// back as written.
type tagAddresses struct{}

func (tagAddresses) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	s := sessionOf(pc)
	if s == nil {
		return
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || s.err != nil {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Link:
			node.Destination, node.Title = s.inText(node.Destination), s.inText(node.Title)
		case *ast.Image:
			node.Destination, node.Title = s.inText(node.Destination), s.inText(node.Title)
		}
		return ast.WalkContinue, nil
	})
}

// inText draws the tags in text that is not markdown, as plain text.
func (s *session) inText(b []byte) []byte {
	if !bytes.Contains(b, []byte(tokenStart)) {
		return b
	}
	var out []byte
	for len(b) > 0 {
		i := bytes.Index(b, []byte(tokenStart))
		if i < 0 {
			break
		}
		out = append(out, b[:i]...)
		n, width, ok := tokenAt(b[i:])
		if !ok || n >= len(s.tokens) {
			out = append(out, b[i:i+len(tokenStart)]...)
			b = b[i+len(tokenStart):]
			continue
		}
		b = b[i+width:]
		tok := s.tokens[n]
		switch {
		case s.draw == nil:
			out = append(out, tok.text...)
		case tok.role != roleSingle:
			s.fail(tok.call, "a shortcode with a closing tag cannot stand in a link's address")
			return out
		default:
			drawn, err := s.drawCall(tok.call)
			if err != nil {
				if s.err == nil {
					s.err = err
				}
				return out
			}
			out = append(out, strings.TrimSpace(stdhtml.UnescapeString(drawn))...)
		}
	}
	return append(out, b...)
}

// errNoShortcodes is the reply of a renderer given no shortcodes to draw with.
var errNoShortcodes = errors.New("not defined")

// drawCall draws one call. A failure is reported at the call that failed, so
// a call inside another is reported at its own line.
func (s *session) drawCall(c *Call) (string, error) {
	if s.draw == nil {
		return "", &ShortcodeError{Line: c.Line, Name: c.Name, Err: errNoShortcodes}
	}
	out, err := s.draw.Draw(c)
	if err != nil {
		if inner, ok := errors.AsType[*ShortcodeError](err); ok {
			return "", inner
		}
		return "", &ShortcodeError{Line: c.Line, Name: c.Name, Err: err}
	}
	return out, nil
}

// tagRenderer draws the tags of the tree.
type tagRenderer struct{}

func (tagRenderer) RegisterFuncs(r renderer.NodeRendererFuncRegisterer) {
	r.Register(kindTagBlock, renderTag)
	r.Register(kindTagInline, renderTag)
}

func renderTag(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	t := n.(tagNode)
	s := t.session()
	tok := s.tokens[t.number()]
	c := tok.call
	if tok.role == roleClose {
		return ast.WalkSkipChildren, nil // paired and dropped, or reported
	}
	if tok.role == roleOpen {
		c.inner = func() (string, error) {
			var buf bytes.Buffer
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if err := s.render(&buf, child); err != nil {
					return "", err
				}
			}
			return s.expand(buf.String())
		}
	}
	out, err := s.drawCall(c)
	if err != nil {
		return ast.WalkStop, err
	}
	t.hide(!c.shown)
	_, _ = w.WriteString(out)
	if _, block := n.(*tagBlock); block && !strings.HasSuffix(out, "\n") {
		_ = w.WriteByte('\n')
	}
	return ast.WalkSkipChildren, nil
}

// expand draws the tags still in rendered HTML: those in raw HTML, which the
// parser leaves as text. A pair there encloses the HTML between its tags.
func (s *session) expand(html string) (string, error) {
	if !strings.Contains(html, tokenStart) {
		return html, nil
	}
	var b strings.Builder
	for {
		i := strings.Index(html, tokenStart)
		if i < 0 {
			b.WriteString(html)
			return b.String(), nil
		}
		n, width, ok := tokenAt(html[i:])
		if !ok || n >= len(s.tokens) {
			b.WriteString(html[:i+len(tokenStart)])
			html = html[i+len(tokenStart):]
			continue
		}
		b.WriteString(html[:i])
		html = html[i+width:]
		tok := s.tokens[n]
		c := tok.call
		switch tok.role {
		case roleClose:
			return "", &ShortcodeError{Line: tok.line, Name: c.Name,
				Err: fmt.Errorf("the closing tag has no opening tag before it in the same HTML")}
		case roleOpen:
			shut := tokenOf(s.closes[c])
			j := strings.Index(html, shut)
			if j < 0 {
				return "", &ShortcodeError{Line: c.Line, Name: c.Name,
					Err: fmt.Errorf("its closing tag on line %d is not in the same HTML", s.tokens[s.closes[c]].line)}
			}
			inner := html[:j]
			c.inner = func() (string, error) { return s.expand(inner) }
			html = html[j+len(shut):]
		}
		out, err := s.drawCall(c)
		if err != nil {
			return "", err
		}
		b.WriteString(out)
	}
}

// withoutTokens drops the tokens left in text: those of tags a backslash kept
// from being read, which are drawn once the page is written.
func withoutTokens(b []byte) []byte {
	if !bytes.Contains(b, []byte(tokenStart)) {
		return b
	}
	var out []byte
	for {
		i := bytes.Index(b, []byte(tokenStart))
		if i < 0 {
			return append(out, b...)
		}
		out = append(out, b[:i]...)
		if _, width, ok := tokenAt(b[i:]); ok {
			b = b[i+width:]
		} else {
			out = append(out, tokenStart...)
			b = b[i+len(tokenStart):]
		}
	}
}

// hiddenTag reports whether n is a tag whose inner content the page does not
// show.
func hiddenTag(n ast.Node) bool {
	t, ok := n.(tagNode)
	return ok && t.hidden()
}
