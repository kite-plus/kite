// Package url generates every link a site emits.
//
// A static build serves directory indexes while a dynamic server matches
// routes, so the two will drift apart the moment anything builds a link by
// concatenating strings. Every permalink, pagination link and taxonomy link in
// Kite comes from this one resolver, and themes are forbidden from assembling
// paths themselves.
package url

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/kite-plus/kite/internal/content"
)

// Style controls how a path is written out.
type Style string

const (
	// StyleDirectory emits /posts/hello/ and writes posts/hello/index.html.
	// This keeps URLs identical between a static host and a dynamic server.
	StyleDirectory Style = "directory"

	// StyleExtension emits /posts/hello.html for hosts without directory
	// indexes.
	StyleExtension Style = "extension"
)

// Options configures the resolver.
type Options struct {
	BaseURL string
	Style   Style

	// PaginationPath is the segment inserted before a page number.
	PaginationPath string

	// TaxonomyRoute is the pattern for a taxonomy listing, and TermRoute for a
	// single term. Both accept :taxonomy and :term.
	TaxonomyRoute string
	TermRoute     string

	// DefaultLocale is emitted without a language prefix.
	DefaultLocale string

	// LocalePrefix adds /<locale> in front of non-default locales.
	//
	// The multilingual URL strategy has to be fixed before the theme contract
	// freezes because it changes the permalink of every page on the site.
	LocalePrefix bool
}

// DefaultOptions returns the resolver defaults.
func DefaultOptions() Options {
	return Options{
		Style:          StyleDirectory,
		PaginationPath: "page",
		TaxonomyRoute:  "/:taxonomy",
		TermRoute:      "/:taxonomy/:term",
		LocalePrefix:   true,
	}
}

// Resolver builds links.
type Resolver struct {
	opts  Options
	types *content.Registry
	base  *url.URL
}

// New returns a resolver. An unparsable base URL is an error rather than a
// silently relative site.
func New(opts Options, types *content.Registry) (*Resolver, error) {
	if opts.Style == "" {
		opts.Style = StyleDirectory
	}
	if opts.PaginationPath == "" {
		opts.PaginationPath = "page"
	}

	r := &Resolver{opts: opts, types: types}
	if opts.BaseURL != "" {
		u, err := url.Parse(opts.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("url: invalid baseURL %q: %w", opts.BaseURL, err)
		}
		r.base = u
	}
	return r, nil
}

// For returns the site-relative URL of an item.
func (r *Resolver) For(item *content.Content) string {
	t := r.types.Get(item.Kind)
	if t == nil {
		return r.finish(item.Locale, "/"+item.Slug)
	}
	return r.finish(item.Locale, expand(t.Route, map[string]string{
		"slug": item.Slug,
		"kind": string(item.Kind),
	}))
}

// ForSummary is [Resolver.For] for a list projection.
func (r *Resolver) ForSummary(s content.Summary) string {
	t := r.types.Get(s.Kind)
	if t == nil {
		return r.finish(s.Locale, "/"+s.Slug)
	}
	return r.finish(s.Locale, expand(t.Route, map[string]string{
		"slug": s.Slug,
		"kind": string(s.Kind),
	}))
}

// ForList returns the URL of a kind's listing page.
func (r *Resolver) ForList(kind content.Kind, locale string) string {
	t := r.types.Get(kind)
	if t == nil {
		return r.finish(locale, "/"+string(kind))
	}
	return r.finish(locale, "/"+t.Dir)
}

// ForTaxonomy returns the URL listing every term of a taxonomy.
func (r *Resolver) ForTaxonomy(taxonomy, locale string) string {
	return r.finish(locale, expand(r.opts.TaxonomyRoute, map[string]string{
		"taxonomy": slugSegment(taxonomy),
	}))
}

// ForTerm returns the URL of one term's listing.
func (r *Resolver) ForTerm(taxonomy, term, locale string) string {
	return r.finish(locale, expand(r.opts.TermRoute, map[string]string{
		"taxonomy": slugSegment(taxonomy),
		"term":     slugSegment(term),
	}))
}

// ForPage returns page n of a listing. Page 1 is the listing itself, so that
// the first page has exactly one URL.
func (r *Resolver) ForPage(base string, n int) string {
	if n <= 1 {
		return base
	}
	trimmed := strings.TrimSuffix(base, "/")
	if r.opts.Style == StyleExtension {
		trimmed = strings.TrimSuffix(trimmed, ".html")
	}
	return r.decorate(trimmed + "/" + r.opts.PaginationPath + "/" + strconv.Itoa(n))
}

// Absolute turns a site-relative URL into an absolute one.
func (r *Resolver) Absolute(rel string) string {
	if r.base == nil {
		return rel
	}
	ref, err := url.Parse(rel)
	if err != nil {
		return rel
	}
	return r.base.ResolveReference(ref).String()
}

// OutputPath returns the file a site-relative URL is written to during a
// static build. It is the inverse of the URL style, which is why both live
// here: a build that guessed this separately would drift from the links.
func (r *Resolver) OutputPath(rel string) string {
	trimmed := strings.Trim(rel, "/")
	if trimmed == "" {
		return "index.html"
	}
	if r.opts.Style == StyleExtension {
		if strings.HasSuffix(trimmed, ".html") {
			return trimmed
		}
		return trimmed + ".html"
	}
	return trimmed + "/index.html"
}

// finish applies the locale prefix and the URL style.
func (r *Resolver) finish(locale, path string) string {
	if r.opts.LocalePrefix && locale != "" && locale != r.opts.DefaultLocale {
		path = "/" + locale + path
	}
	return r.decorate(path)
}

func (r *Resolver) decorate(path string) string {
	path = "/" + strings.Trim(path, "/")
	if path == "/" {
		return "/"
	}
	if r.opts.Style == StyleExtension {
		return path + ".html"
	}
	return path + "/"
}

// expand substitutes :name placeholders in a route pattern.
func expand(pattern string, values map[string]string) string {
	if pattern == "" {
		return "/"
	}
	segments := strings.Split(strings.Trim(pattern, "/"), "/")
	out := make([]string, 0, len(segments))
	for _, seg := range segments {
		if after, ok := strings.CutPrefix(seg, ":"); ok {
			if v, found := values[after]; found {
				out = append(out, pathEscape(v))
				continue
			}
			continue
		}
		out = append(out, seg)
	}
	return "/" + strings.Join(out, "/")
}

// slugSegment normalizes a taxonomy or term name for use in a path. Terms are
// free text, so "Web Dev" has to become a single usable segment.
func slugSegment(s string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.TrimSpace(s) {
		switch r {
		case '/', ' ', '\t':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			b.WriteRune(toLower(r))
			lastDash = false
		}
	}
	return strings.Trim(b.String(), "-")
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// pathEscape escapes only what a path segment must escape.
//
// url.PathEscape percent-encodes every non-ASCII byte, which would turn a
// Chinese slug into an unreadable string in the address bar and in the
// generated HTML. Letters and digits of any script are left alone; everything
// a URL treats as reserved or unsafe is encoded.
func pathEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if safeInPath(r) {
			b.WriteRune(r)
			continue
		}
		for _, c := range []byte(string(r)) {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// safeInPath reports whether a rune may appear literally in a path segment.
func safeInPath(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case strings.ContainsRune("-._~!$&'()*+,;=:@", r):
		return true
	case r > unicode.MaxASCII:
		return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
	default:
		return false
	}
}
