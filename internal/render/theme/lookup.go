package theme

import (
	"io/fs"
	"strings"
)

// Target names the template wanted for one page.
type Target struct {
	// Kind is home, single, list, taxonomy, term or 404.
	Kind string

	// Type is the content type, or the taxonomy name for taxonomy and term
	// pages. It is the optional first directory level.
	Type string

	// Layout overrides the kind, from a page's front matter.
	Layout string

	// Format is the output format: "html" is default and carries no infix,
	// others produce names such as single.rss.xml.
	Format string

	// Lang is an optional language suffix.
	Lang string
}

// Format describes one output format.
type Format struct {
	Name      string
	MediaType string
	Extension string
	// Infix is inserted before the extension for non-default formats.
	Infix string
}

// FormatHTML is the default output format.
var FormatHTML = Format{Name: "html", MediaType: "text/html; charset=utf-8", Extension: "html"}

// Candidate is one possible template together with where it was found.
type Candidate struct {
	Path   string
	Source string
}

// Candidates returns the lookup chain for a target, most specific first.
//
// Only four dimensions are consulted: kind, type, layout and output format,
// plus a language suffix. Section-path recursion is deliberately absent: it is
// the part of Hugo's rules that cannot be explained to a theme author, and a
// lookup order nobody can predict is a lookup order nobody can override
// correctly.
func Candidates(t Target, formats map[string]Format) []string {
	format, ok := formats[t.Format]
	if !ok {
		format = FormatHTML
	}

	// Base names, most specific first: an explicit layout beats the kind.
	var bases []string
	if t.Layout != "" {
		bases = append(bases, t.Layout)
	}
	if t.Kind != "" {
		bases = append(bases, t.Kind)
	}
	if len(bases) == 0 {
		bases = []string{"single"}
	}

	// Directory prefixes, most specific first.
	prefixes := []string{""}
	if t.Type != "" {
		prefixes = []string{t.Type + "/", ""}
	}

	var out []string
	for _, prefix := range prefixes {
		for _, base := range bases {
			if t.Lang != "" {
				out = append(out, prefix+filename(base, t.Lang, format))
			}
			out = append(out, prefix+filename(base, "", format))
		}
	}
	return out
}

// filename assembles base[.lang][.infix].ext.
func filename(base, lang string, f Format) string {
	var b strings.Builder
	b.WriteString(base)
	if f.Infix != "" {
		b.WriteByte('.')
		b.WriteString(f.Infix)
	}
	if lang != "" {
		b.WriteByte('.')
		b.WriteString(lang)
	}
	b.WriteByte('.')
	b.WriteString(f.Extension)
	return b.String()
}

// reservedPrefix marks directories under layouts/ that are not routable pages.
const reservedPrefix = "_"

// IsReserved reports whether a path sits in a reserved directory such as
// _partials.
func IsReserved(path string) bool {
	for seg := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(seg, reservedPrefix) {
			return true
		}
	}
	return false
}

// resolve walks the lookup chain and returns the first template that exists.
//
// Candidates are tried in order of specificity, and each candidate is looked
// for in every source before moving on. Trying all site templates before any
// theme template would let a site's generic list.html shadow a theme's
// carefully written post/list.html, which is never what the user means.
func resolve(sources []Source, candidates []string) (Candidate, bool) {
	for _, name := range candidates {
		for _, src := range sources {
			if exists(src.FS, name) {
				return Candidate{Path: name, Source: src.Name}, true
			}
		}
	}
	return Candidate{}, false
}

func exists(fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	return err == nil && !info.IsDir()
}
