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
	for _, style := range []kurl.Style{kurl.StyleDirectory, kurl.StyleExtension} {
		t.Run(string(style), func(t *testing.T) {
			r := newResolver(t, func(o *kurl.Options) { o.Style = style })

			cases := []string{
				r.For(post("hello")),
				r.ForList("post", ""),
				r.ForTaxonomy("tag", ""),
				r.ForTerm("tag", "Go", ""),
				r.ForPage(r.ForList("post", ""), 3),
				"/",
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
				if got := servePath(out, style); got != link {
					t.Errorf("round trip broke: link %q wrote %q which serves as %q", link, out, got)
				}
			}
		})
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
	if got := r.ForTerm("tag", "Web Dev", ""); got != "/tag/web-dev/" {
		t.Errorf("ForTerm = %q", got)
	}
	// A term containing a slash must not create an extra path level.
	if got := r.ForTerm("tag", "a/b", ""); got != "/tag/a-b/" {
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
