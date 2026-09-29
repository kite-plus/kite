package content

import (
	"cmp"
	"maps"
	"slices"
	"strings"
)

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

// TermSpellings lists the ways items write one term, given how many items
// write it each way: the way most of them do first, and ways used alike in
// byte order, so Go before go. The first is what the term is called.
func TermSpellings(spellings map[string]int) []string {
	return slices.SortedFunc(maps.Keys(spellings), func(a, b string) int {
		return cmp.Or(cmp.Compare(spellings[b], spellings[a]), strings.Compare(a, b))
	})
}

// TermName is what a term written more than one way is called: the first of
// its [TermSpellings].
func TermName(spellings map[string]int) string {
	if names := TermSpellings(spellings); len(names) > 0 {
		return names[0]
	}
	return ""
}
