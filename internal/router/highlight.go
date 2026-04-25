package router

import (
	"bytes"
	"io"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// shareTextPreviewCap is the upper bound for how much of a text file we will
// render in-line on the share page. Mirrors the previous client-side limit so
// a 200MB log doesn't lock the tab.
const shareTextPreviewCap = 256 * 1024

// chromaStyle is the syntax-highlight palette baked into the share-page CSS.
// `github` reads cleanly on the page's light background.
var chromaStyle = func() *chroma.Style {
	if s := styles.Get("github"); s != nil {
		return s
	}
	return styles.Fallback
}()

// chromaFormatter emits HTML with class names so the colour palette lives in
// a single stylesheet (shareHighlightCSS) rather than being inlined into every
// span.
var chromaFormatter = chromahtml.New(
	chromahtml.WithClasses(true),
	chromahtml.TabWidth(4),
)

// shareHighlightCSS is the chroma stylesheet, generated once at boot. Embedded
// in share.html via `{{.HighlightCSS}}` so the formatter's class names resolve.
var shareHighlightCSS = func() string {
	var buf bytes.Buffer
	if err := chromaFormatter.WriteCSS(&buf, chromaStyle); err != nil {
		return ""
	}
	return buf.String()
}()

// highlightShareText reads up to shareTextPreviewCap bytes from r, picks a
// lexer by filename / mime / content sniffing, and returns chroma-rendered
// HTML. `truncated` is true when the file was longer than the cap so the
// template can show a "showing first N KB" hint.
func highlightShareText(r io.Reader, filename, mime string) (html string, truncated bool, err error) {
	capped := io.LimitReader(r, shareTextPreviewCap+1)
	raw, err := io.ReadAll(capped)
	if err != nil {
		return "", false, err
	}
	if len(raw) > shareTextPreviewCap {
		raw = raw[:shareTextPreviewCap]
		truncated = true
	}

	lexer := pickShareLexer(filename, mime, string(raw))
	iterator, err := lexer.Tokenise(nil, string(raw))
	if err != nil {
		return "", truncated, err
	}
	var buf bytes.Buffer
	if err := chromaFormatter.Format(&buf, chromaStyle, iterator); err != nil {
		return "", truncated, err
	}
	return buf.String(), truncated, nil
}

// pickShareLexer falls through filename → mime → content analysis, then
// finally the plaintext fallback. Mirrors how chroma's own helpers compose
// the same checks but lets us skip the expensive analyse step when an
// earlier match wins.
func pickShareLexer(filename, mime, content string) chroma.Lexer {
	if filename != "" {
		if l := lexers.Match(filename); l != nil {
			return l
		}
	}
	if mime != "" {
		if l := lexers.MatchMimeType(mime); l != nil {
			return l
		}
	}
	if l := lexers.Analyse(content); l != nil {
		return l
	}
	return lexers.Fallback
}
