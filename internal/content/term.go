package content

import "strings"

// TermSlug is a term as its address writes it: trimmed, with A to Z in lower
// case and each run of spaces, tabs and slashes one dash, so "Web Dev" is
// web-dev. Terms with one slug share an address, which makes them one term
// however each item writes it: Go and go are both /tags/go/.
func TermSlug(term string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.TrimSpace(term) {
		switch r {
		case '/', ' ', '\t':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			b.WriteRune(r)
			lastDash = false
		}
	}
	return strings.Trim(b.String(), "-")
}

// TermName is what a term written more than one way is called, given how
// many items write it each way: the way most of them do, and of ways used
// alike, the first in byte order, so Go before go.
func TermName(spellings map[string]int) string {
	var name string
	most := 0
	for spelling, n := range spellings {
		if n > most || n == most && spelling < name {
			name, most = spelling, n
		}
	}
	return name
}
