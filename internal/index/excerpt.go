package index

import (
	"strings"
	"unicode"
)

// excerptLimit is how much prose a summary keeps.
const excerptLimit = 220

// summarize picks the best short description available for an item.
//
// An author-written description always wins: it was written to be read on its
// own, which is more than can be said for the opening of an article.
func summarize(description, body string) string {
	// A description is prose the author wrote, but it is still markdown: the
	// excerpt column is plain text, and it also ends up in a meta tag where
	// emphasis markers would be shown literally.
	if d := inlineText(strings.TrimSpace(description)); d != "" {
		return truncate(d, excerptLimit)
	}
	return truncate(plainProse(body), excerptLimit)
}

// plainProse reduces a markdown body to the readable text of its opening
// paragraphs.
//
// Truncating the raw source instead would put "## Heading" and the pipes of a
// table into the middle of a sentence, which is what a summary is supposed to
// spare the reader.
func plainProse(body string) string {
	var out []string
	inFence := false

	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || skipLine(trimmed) {
			continue
		}

		if text := inlineText(trimmed); text != "" {
			out = append(out, text)
		}
	}
	return strings.Join(out, " ")
}

// skipLine reports whether a line carries structure rather than prose.
func skipLine(trimmed string) bool {
	switch {
	case trimmed == "":
		return true
	case strings.HasPrefix(trimmed, "#"): // heading
		return true
	case strings.HasPrefix(trimmed, "|"): // table row
		return true
	case strings.HasPrefix(trimmed, ">"): // block quote marker
		return true
	case strings.HasPrefix(trimmed, "<"): // raw html
		return true
	case strings.HasPrefix(trimmed, "["): // link reference definition
		return true
	case isRule(trimmed):
		return true
	default:
		return false
	}
}

// isRule reports whether a line is a thematic break such as --- or ***.
func isRule(s string) bool {
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' && c != '=' {
		return false
	}
	return strings.Trim(s, string(c)) == ""
}

// scanState is where the scanner is within a bracketed construct.
type scanState int

const (
	scanProse        scanState = iota
	scanBracket                // inside [...]
	scanAfterBracket           // just past ], a ( would start a destination
	scanDestination            // inside (...)
)

// inlineText strips the markers markdown uses inside a line, keeping the words.
//
// A link keeps its text and loses its destination; an image loses both,
// because alt text describes a picture the summary is not showing.
func inlineText(line string) string {
	line = stripListMarker(line)

	var out, bracket strings.Builder
	state := scanProse
	isImage := false

	flush := func() {
		if !isImage {
			out.WriteString(bracket.String())
		}
		bracket.Reset()
		isImage = false
	}

	for i := 0; i < len(line); i++ {
		c := line[i]

		switch state {
		case scanBracket:
			if c == ']' {
				state = scanAfterBracket
				continue
			}
			bracket.WriteByte(c)
			continue

		case scanAfterBracket:
			if c == '(' {
				state = scanDestination
				continue
			}
			// No destination followed, so the brackets held ordinary text.
			flush()
			state = scanProse

		case scanDestination:
			if c == ')' {
				flush()
				state = scanProse
			}
			continue
		}

		switch {
		case c == '`' || c == '*' || c == '_' || c == '~':
		case c == '!' && i+1 < len(line) && line[i+1] == '[':
			i++
			state, isImage = scanBracket, true
		case c == '[':
			state, isImage = scanBracket, false
		default:
			out.WriteByte(c)
		}
	}
	if state != scanProse {
		flush()
	}

	return strings.TrimSpace(collapseSpace(out.String()))
}

// stripListMarker removes a leading bullet or number so that a summary reads
// as a sentence rather than as a fragment of a list.
func stripListMarker(line string) string {
	rest := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(rest, "- "), strings.HasPrefix(rest, "* "), strings.HasPrefix(rest, "+ "):
		return rest[2:]
	}

	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits+1 < len(rest) && (rest[digits] == '.' || rest[digits] == ')') && rest[digits+1] == ' ' {
		return rest[digits+2:]
	}
	return rest
}

func collapseSpace(s string) string {
	var b strings.Builder
	space := true
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		b.WriteRune(r)
		space = false
	}
	return b.String()
}

// truncate cuts at a word boundary where one is near, so a summary does not
// end mid-word.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	cut := string(runes[:limit])
	if i := strings.LastIndexByte(cut, ' '); i > limit/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}
