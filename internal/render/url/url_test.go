package url_test

import (
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/content"
	kurl "github.com/kite-plus/kite/internal/render/url"
)

func newResolver(t *testing.T, mutate func(*kurl.Options)) *kurl.Resolver {
	t.Helper()
	opts := kurl.DefaultOptions()
	opts.BaseURL = "https://example.com"
	if mutate != nil {
		mutate(&opts)
	}
	r, err := kurl.New(opts, content.DefaultRegistry())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func post(slug string) *content.Content {
	return &content.Content{Kind: "post", Slug: slug, Title: slug}
}

func TestPermalinkFollowsTheContentTypeRoute(t *testing.T) {
	r := newResolver(t, nil)
	if got := r.For(post("hello-world")); got != "/posts/hello-world/" {
		t.Errorf("For = %q", got)
	}
	page := &content.Content{Kind: "page", Slug: "about"}
	if got := r.For(page); got != "/about/" {
		t.Errorf("page For = %q", got)
	}
}

func TestExtensionStyle(t *testing.T) {
	r := newResolver(t, func(o *kurl.Options) { o.Style = kurl.StyleExtension })
	if got := r.For(post("hello")); got != "/posts/hello.html" {
		t.Errorf("For = %q", got)
	}
}

// A link and the file a static build writes for it must be two views of one
// decision. If they can disagree, a static site links to pages it never wrote.
func TestOutputPathIsTheInverseOfTheURL(t *testing.T) {
	// The host publishes the output directory at the base URL's path.
	bases := map[string]string{"https://example.com": "", "https://example.github.io/blog/": "/blog"}
	for base, prefix := range bases {
		for _, style := range []kurl.Style{kurl.StyleDirectory, kurl.StyleExtension} {
			t.Run(prefix+"/"+string(style), func(t *testing.T) {
				r := newResolver(t, func(o *kurl.Options) {
					o.BaseURL = base
					o.Style = style
				})

				cases := []string{
					r.For(post("hello")),
					r.ForList("post", ""),
					r.ForTaxonomy("tags", ""),
					r.ForTerm("tags", "Go", ""),
					r.ForPage(r.ForList("post", ""), 3),
					r.ForHome(""),
					r.ForPage(r.ForHome(""), 2),
				}
				for _, link := range cases {
					out := r.OutputPath(link)
					if strings.HasPrefix(out, "/") {
						t.Errorf("%s: output path must be relative, got %q", link, out)
					}
					if !strings.HasSuffix(out, ".html") {
						t.Errorf("%s: output path must be an html file, got %q", link, out)
					}
					// Serving the written file at that path must reproduce the link.
					if got := prefix + servePath(out, style); got != link {
						t.Errorf("round trip broke: link %q wrote %q which serves as %q", link, out, got)
					}
				}
			})
		}
	}
}

// servePath models what a static host does with a written file. Every host
// serves index.html at the directory it sits in, whatever the URL style.
func servePath(out string, style kurl.Style) string {
	if trimmed, ok := strings.CutSuffix(out, "index.html"); ok {
		return "/" + trimmed
	}
	if style == kurl.StyleExtension {
		return "/" + out
	}
	return "/" + out
}

func TestPaginationFirstPageHasOneURL(t *testing.T) {
	r := newResolver(t, nil)
	base := r.ForList("post", "")
	if got := r.ForPage(base, 1); got != base {
		t.Errorf("page 1 = %q, want the listing itself %q", got, base)
	}
	if got := r.ForPage(base, 0); got != base {
		t.Errorf("page 0 = %q, want %q", got, base)
	}
	if got := r.ForPage(base, 2); got != "/posts/page/2/" {
		t.Errorf("page 2 = %q", got)
	}
}

func TestTermSegmentsAreNormalized(t *testing.T) {
	r := newResolver(t, nil)
	if got := r.ForTerm("tags", "Web Dev", ""); got != "/tags/web-dev/" {
		t.Errorf("ForTerm = %q", got)
	}
	// A term containing a slash must not create an extra path level.
	if got := r.ForTerm("tags", "a/b", ""); got != "/tags/a-b/" {
		t.Errorf("ForTerm with slash = %q", got)
	}
}

func TestLocalePrefix(t *testing.T) {
	r := newResolver(t, func(o *kurl.Options) { o.DefaultLocale = "en" })

	item := post("hello")
	item.Locale = "en"
	if got := r.For(item); got != "/posts/hello/" {
		t.Errorf("default locale should carry no prefix, got %q", got)
	}

	item.Locale = "zh-CN"
	if got := r.For(item); got != "/zh-CN/posts/hello/" {
		t.Errorf("non-default locale = %q", got)
	}
}

func TestAbsolute(t *testing.T) {
	r := newResolver(t, nil)
	if got := r.Absolute("/posts/hello/"); got != "https://example.com/posts/hello/" {
		t.Errorf("Absolute = %q", got)
	}
}

// A GitHub Pages project site without a domain lives under the repository's
// name, and every link it makes has to stay under it.
func TestABasePathPrefixesEveryLink(t *testing.T) {
	for _, base := range []string{"https://example.github.io/blog/", "https://example.github.io/blog"} {
		t.Run(base, func(t *testing.T) {
			r := newResolver(t, func(o *kurl.Options) {
				o.BaseURL = base
				o.DefaultLocale = "en"
			})
			item := post("hello")
			item.Locale = "zh-CN"
			for _, c := range [][2]string{
				{r.For(post("hello")), "/blog/posts/hello/"},
				{r.For(item), "/blog/zh-CN/posts/hello/"},
				{r.ForHome("en"), "/blog/"},
				{r.ForPage(r.ForHome("en"), 2), "/blog/page/2/"},
				{r.ForList("post", "en"), "/blog/posts/"},
				{r.ForPage(r.ForList("post", "en"), 2), "/blog/posts/page/2/"},
				{r.ForTaxonomy("tags", "en"), "/blog/tags/"},
				{r.ForTerm("tags", "Go", "en"), "/blog/tags/go/"},
				{r.Absolute(r.For(post("hello"))), "https://example.github.io/blog/posts/hello/"},
			} {
				if c[0] != c[1] {
					t.Errorf("got %q, want %q", c[0], c[1])
				}
			}
		})
	}
}

func TestABasePathUnderTheExtensionStyle(t *testing.T) {
	r := newResolver(t, func(o *kurl.Options) {
		o.BaseURL = "https://example.github.io/blog/"
		o.Style = kurl.StyleExtension
	})
	for _, c := range [][2]string{
		{r.For(post("hello")), "/blog/posts/hello.html"},
		{r.ForHome(""), "/blog/"},
		{r.ForPage(r.ForHome(""), 2), "/blog/page/2.html"},
		{r.ForList("post", ""), "/blog/posts.html"},
		{r.ForPage(r.ForList("post", ""), 2), "/blog/posts/page/2.html"},
	} {
		if c[0] != c[1] {
			t.Errorf("got %q, want %q", c[0], c[1])
		}
	}
}

func TestRelLinksAPathWithinTheSite(t *testing.T) {
	sub := newResolver(t, func(o *kurl.Options) { o.BaseURL = "https://example.github.io/blog/" })
	root := newResolver(t, nil)

	for _, c := range []struct {
		r        *kurl.Resolver
		in, want string
	}{
		{sub, "rss.xml", "/blog/rss.xml"},
		{sub, "/rss.xml", "/blog/rss.xml"},
		{sub, "/about/", "/blog/about/"},
		{sub, "", "/blog/"},
		{sub, "/", "/blog/"},
		{root, "rss.xml", "/rss.xml"},
		{root, "/", "/"},
		// References that leave the site's paths are not the resolver's to
		// change.
		{sub, "https://github.com/kite-plus/kite", "https://github.com/kite-plus/kite"},
		{sub, "//cdn.example.com/font.css", "//cdn.example.com/font.css"},
		{sub, "mailto:someone@example.com", "mailto:someone@example.com"},
		{sub, "#main", "#main"},
	} {
		if got := c.r.Rel(c.in); got != c.want {
			t.Errorf("Rel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSitePathRemovesTheBasePath(t *testing.T) {
	r := newResolver(t, func(o *kurl.Options) { o.BaseURL = "https://example.github.io/blog/" })
	for in, want := range map[string]string{
		"/blog/posts/hello/": "/posts/hello/",
		"/blog/rss.xml":      "/rss.xml",
		"/blog/":             "/",
		"/blog":              "/",
	} {
		if got, ok := r.SitePath(in); !ok || got != want {
			t.Errorf("SitePath(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, outside := range []string{"/", "/rss.xml", "/blogger/", "/other/blog/"} {
		if got, ok := r.SitePath(outside); ok {
			t.Errorf("SitePath(%q) = %q, want it reported outside the site", outside, got)
		}
	}

	root := newResolver(t, nil)
	if got, ok := root.SitePath("/posts/hello/"); !ok || got != "/posts/hello/" {
		t.Errorf("a site at the root owns every path; got %q, %v", got, ok)
	}
}

func TestNamedPages(t *testing.T) {
	r := newResolver(t, func(o *kurl.Options) { o.BaseURL = "https://example.github.io/blog/" })
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"home"}, "/blog/"},
		{[]string{"list", "post"}, "/blog/posts/"},
		{[]string{"taxonomy", "tags"}, "/blog/tags/"},
		{[]string{"term", "tags", "Web Dev"}, "/blog/tags/web-dev/"},
	} {
		got, err := r.Named(c.args[0], c.args[1:]...)
		if err != nil || got != c.want {
			t.Errorf("Named(%q) = %q, %v; want %q", c.args, got, err, c.want)
		}
	}
	for _, bad := range [][]string{{"feed"}, {"list"}, {"term", "tags"}, {"home", "extra"}} {
		if got, err := r.Named(bad[0], bad[1:]...); err == nil {
			t.Errorf("Named(%q) = %q, want an error", bad, got)
		}
	}
}

func TestNonASCIISlugStaysReadable(t *testing.T) {
	r := newResolver(t, nil)
	got := r.For(post("你好世界"))
	if strings.Contains(got, "%") {
		t.Errorf("a non-ASCII slug should stay readable, got %q", got)
	}
	if got != "/posts/你好世界/" {
		t.Errorf("For = %q", got)
	}
}

func TestInvalidBaseURLIsRejected(t *testing.T) {
	opts := kurl.DefaultOptions()
	opts.BaseURL = "://nonsense"
	if _, err := kurl.New(opts, content.DefaultRegistry()); err == nil {
		t.Fatal("expected an error for an unparsable base URL")
	}
}
