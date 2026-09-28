package build

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"strings"

	"github.com/kite-plus/kite/internal/render"
)

// planAliases adds a page at each address an item lists under aliases in its
// front matter, where it was once published, which sends a reader on to where
// it is now.
//
// An alias is a path from the site's root or, without a leading slash, one
// beside the item's own address, as Hugo reads it. An alias that is the item's
// own address is left out, since the page itself answers there. One that is
// another page's address, or another alias's, is refused: publishing both
// would leave one of them unreachable, and nothing would say which.
func (b *Builder) planAliases(p *Plan) error {
	taken := make(map[string]string, len(p.Targets))
	for _, t := range p.Targets {
		taken[Address(t.URL)] = describeTarget(t)
	}

	var aliases []Target
	for _, t := range p.Targets {
		if t.Kind != render.KindSingle || t.Item == nil {
			continue
		}
		own, _ := b.opts.Resolver.SitePath(t.URL)
		for _, alias := range t.Item.Aliases {
			site, err := aliasPath(own, alias)
			if err != nil {
				return fmt.Errorf("build: %s: %w", t.Item.Locator, err)
			}
			link := b.opts.Resolver.Rel(site)
			key := Address(link)
			if key == Address(t.URL) {
				continue
			}
			if held, ok := taken[key]; ok {
				return fmt.Errorf("build: %s: the alias %s is the address of %s; take it out of one of them",
					t.Item.Locator, alias, held)
			}
			taken[key] = "an alias of " + string(t.Item.Locator)
			aliases = append(aliases, Target{
				Kind:     render.KindAlias,
				URL:      link,
				Path:     aliasFile(site),
				Type:     t.Type,
				Title:    t.Item.Title,
				Redirect: t.URL,
			})
		}
	}
	p.Targets = append(p.Targets, aliases...)
	return nil
}

func describeTarget(t Target) string {
	if t.Item != nil {
		return string(t.Item.Locator)
	}
	return "the page at " + t.URL
}

// aliasPath reads an alias as a path within the site, relative to the item's
// own address own, and keeps the trailing slash of one that names a folder.
func aliasPath(own, alias string) (string, error) {
	a := strings.TrimSpace(alias)
	if decoded, err := url.PathUnescape(a); err == nil {
		a = decoded
	}
	switch {
	case a == "":
		return "", fmt.Errorf("an alias is empty")
	case strings.Contains(a, "://") || strings.HasPrefix(a, "//") || strings.ContainsAny(a, "?#"):
		return "", fmt.Errorf("the alias %q is not a path within the site", alias)
	}
	folder := !isPageFile(a)
	if !strings.HasPrefix(a, "/") {
		a = path.Join(path.Dir(strings.TrimSuffix(own, "/")), a)
	}
	a = path.Clean("/" + a)
	if folder && a != "/" {
		a += "/"
	}
	return a, nil
}

// isPageFile reports whether a path names a page's file rather than a folder
// whose index a host serves.
func isPageFile(p string) bool {
	ext := strings.ToLower(path.Ext(strings.TrimSuffix(p, "/")))
	return !strings.HasSuffix(p, "/") && (ext == ".html" || ext == ".htm")
}

// aliasFile is the file an alias path is written to. A folder gets an index a
// host serves at its address whatever the site's url style, since that is
// how the old address was reached.
func aliasFile(site string) string {
	rel := strings.TrimPrefix(site, "/")
	if rel == "" || strings.HasSuffix(rel, "/") {
		return rel + "index.html"
	}
	return rel
}

// aliasPage is what an alias serves, as Hugo writes one: a browser is sent on
// at once, a search engine is told the address to keep, and anything that
// follows neither has a link. Relative links let a preview stay on the
// machine it runs on. No theme draws it: nobody should have to look at it.
func aliasPage(lang, title, to, canonical string) []byte {
	e := html.EscapeString
	return fmt.Appendf(nil, `<!DOCTYPE html>
<html lang="%s">
<head>
<meta charset="utf-8">
<title>%s</title>
<link rel="canonical" href="%s">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url=%s">
</head>
<body>
<p><a href="%s">%s</a></p>
</body>
</html>
`, e(lang), e(title), e(canonical), e(to), e(to), e(title))
}
