// Package builtin implements Kite's own features as hooks.
//
// These are deliberately not called directly from the build engine. They are
// the first consumers of the hook API, which is how the extension points get
// exercised long before any third party depends on them.
package builtin

import (
	"bytes"
	"cmp"
	"context"
	"encoding/xml"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/stamp"
)

// Register adds every built-in hook to a bus.
func Register(bus *hook.Bus, opts Options) {
	if opts.Sitemap {
		bus.Register(&Sitemap{
			Base: hook.Base{HookName: "sitemap", HookPhase: hook.PhaseBuild, HookVersion: "1"},
		}, hook.DefaultPriority)
	}
	if opts.Feed {
		kinds := opts.FeedKinds
		if len(kinds) == 0 {
			kinds = []string{"post"}
		}
		bus.Register(&Feed{
			Base:      hook.Base{HookName: "feed", HookPhase: hook.PhaseBuild, HookVersion: "1"},
			Limit:     opts.FeedLimit,
			Kinds:     kinds,
			Aliases:   opts.FeedAliases,
			Generator: opts.Generator,
		}, hook.DefaultPriority)
	}
	if opts.Stamp != nil {
		bus.Register(&BuildStamp{
			Base:   hook.Base{HookName: "build-stamp", HookPhase: hook.PhaseBuild, HookVersion: "1"},
			Commit: opts.Stamp,
		}, hook.DefaultPriority)
	}
	if opts.Generator != "" {
		bus.Register(&Generator{
			Base:   hook.Base{HookName: "generator", HookPhase: hook.PhaseBuild, HookVersion: "1"},
			Credit: opts.Generator,
		}, hook.DefaultPriority)
	}
}

// Options selects which built-in hooks to enable.
type Options struct {
	Sitemap   bool
	Feed      bool
	FeedLimit int

	// FeedKinds limits the feed to certain content kinds. A standalone page
	// such as "about" is part of the site but is not news, so it does not
	// belong in a feed readers subscribe to.
	FeedKinds []string

	// FeedAliases are more files the same feed is written to, for readers
	// subscribed at an address the site used before.
	FeedAliases []string

	// Stamp, when set, returns the commit the site is built from, which is
	// written to stamp.File. It is asked at each build, since a server
	// that stays up sees new commits.
	Stamp func() string

	// Generator names the Kite building the site, which every page and the
	// feed credit; empty, nothing does.
	Generator string
}

// DefaultOptions enables the hooks every site wants.
func DefaultOptions() Options {
	return Options{Sitemap: true, Feed: true, FeedLimit: 20, FeedKinds: []string{"post"}}
}

// Sitemap writes sitemap.xml.
type Sitemap struct{ hook.Base }

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapSet struct {
	XMLName xml.Name     `xml:"urlset"`
	NS      string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// BuildComplete emits the sitemap.
func (s *Sitemap) BuildComplete(_ context.Context, b *hook.BuildInfo) error {
	set := sitemapSet{NS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, p := range b.Pages {
		if !p.Indexable {
			continue
		}
		entry := sitemapURL{Loc: absolute(b.Site.BaseURL, p.URL)}
		if p.Item != nil && !p.Item.UpdatedAt.IsZero() {
			entry.LastMod = p.Item.UpdatedAt.UTC().Format(time.RFC3339)
		}
		set.URLs = append(set.URLs, entry)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(set); err != nil {
		return err
	}
	buf.WriteByte('\n')
	return b.Emit("sitemap.xml", buf.Bytes())
}

// Generator names the Kite that built a page in the page's head, whatever
// theme drew it. A page that names a generator already keeps its own.
type Generator struct {
	hook.Base
	Credit string
}

// CacheKey changes with the Kite named, as every page does.
func (g *Generator) CacheKey() []byte { return append(g.Base.CacheKey(), g.Credit...) }

var (
	headEnd   = regexp.MustCompile(`(?i)</head\s*>`)
	generator = regexp.MustCompile(`(?i)<meta\s[^>]*\bname\s*=\s*["']?generator["'\s/>]`)
)

// TransformHTML adds the meta tag before the first </head>. A page that does
// not close its head is left as it is.
func (g *Generator) TransformHTML(_ context.Context, doc *hook.HTMLDoc) error {
	at := headEnd.FindStringIndex(doc.HTML)
	if at == nil || generator.MatchString(doc.HTML[:at[0]]) {
		return nil
	}
	meta := `<meta name="generator" content="` + html.EscapeString(g.Credit) + `">` + "\n"
	doc.HTML = doc.HTML[:at[0]] + meta + doc.HTML[at[0]:]
	return nil
}

// BuildStamp writes the commit the site was built from, for the studio to
// read back from the live site. A site built from no commit it can name gets
// no stamp.
type BuildStamp struct {
	hook.Base
	Commit func() string
}

// BuildComplete emits the stamp.
func (s *BuildStamp) BuildComplete(_ context.Context, b *hook.BuildInfo) error {
	commit := s.Commit()
	if commit == "" {
		return nil
	}
	return b.Emit(stamp.File, stamp.Encode(commit))
}

// Feed writes an RSS 2.0 feed to rss.xml and to each of its aliases.
type Feed struct {
	hook.Base
	Limit     int
	Kinds     []string
	Aliases   []string
	Generator string
}

// CacheKey changes with the Kite the feed credits, as the feed does.
func (f *Feed) CacheKey() []byte { return append(f.Base.CacheKey(), f.Generator...) }

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate,omitempty"`
	Description string  `xml:"description,omitempty"`
}

// rssGUID is an item's id. RSS takes a guid for the item's address unless it
// says otherwise, and an id is not one.
type rssGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	ID          string `xml:",chardata"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language,omitempty"`
	Generator   string    `xml:"generator,omitempty"`
	Updated     string    `xml:"lastBuildDate,omitempty"`
	Self        *atomLink `xml:"atom:link"`
	Items       []rssItem `xml:"item"`
}

// atomLink gives the feed's own address, which RSS has no element for.
type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr,omitempty"`
	Channel rssChannel `xml:"channel"`
}

// BuildComplete emits the feed.
func (f *Feed) BuildComplete(_ context.Context, b *hook.BuildInfo) error {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}

	var entries []hook.PageInfo
	for _, p := range b.Pages {
		if p.Item == nil || !p.Indexable {
			continue // listings and taxonomy pages are not feed entries
		}
		if slices.Contains(f.Kinds, string(p.Item.Kind)) {
			entries = append(entries, p)
		}
	}
	// Pages come in the order of their files, which says nothing of when
	// they were published. The feed takes the site's listing order: newest
	// first, undated last, and the newer id first between items of a date.
	slices.SortFunc(entries, func(x, y hook.PageInfo) int {
		return cmp.Or(published(y).Compare(published(x)), strings.Compare(string(y.Item.ID), string(x.Item.ID)))
	})
	entries = entries[:min(limit, len(entries))]

	channel := rssChannel{
		Title:       b.Site.Title,
		Link:        b.Site.BaseURL,
		Description: b.Site.Description,
		Language:    b.Site.Language,
		Generator:   f.Generator,
	}
	// The feed changed when its newest item was published. The clock would
	// make two builds of one site differ.
	if len(entries) > 0 && entries[0].Item.PublishedAt != nil {
		channel.Updated = entries[0].Item.PublishedAt.UTC().Format(time.RFC1123Z)
	}
	for _, p := range entries {
		item := rssItem{
			Title:       p.Title,
			Link:        absolute(b.Site.BaseURL, p.URL),
			GUID:        rssGUID{ID: string(p.Item.ID)},
			Description: p.Excerpt,
		}
		if p.Item.PublishedAt != nil {
			item.PubDate = p.Item.PublishedAt.UTC().Format(time.RFC1123Z)
		}
		channel.Items = append(channel.Items, item)
	}

	feed := rssFeed{Version: "2.0", Channel: channel}
	for _, name := range append([]string{"rss.xml"}, f.Aliases...) {
		// Each file gives its own address, which is where its readers
		// subscribed; a site with no address has none to give.
		if b.Site.BaseURL != "" {
			feed.Atom = "http://www.w3.org/2005/Atom"
			feed.Channel.Self = &atomLink{Href: under(b.Site.BaseURL, name), Rel: "self", Type: "application/rss+xml"}
		}
		var buf bytes.Buffer
		buf.WriteString(xml.Header)
		enc := xml.NewEncoder(&buf)
		enc.Indent("", "  ")
		if err := enc.Encode(feed); err != nil {
			return err
		}
		buf.WriteByte('\n')
		if err := b.Emit(name, buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// published is when a page's item was published, the zero time if never.
func published(p hook.PageInfo) time.Time {
	if p.Item.PublishedAt == nil {
		return time.Time{}
	}
	return *p.Item.PublishedAt
}

// under is the address of a file of the site, beneath the base URL's path;
// resolving the file against the base would drop the path's last part.
func under(base, file string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(file, "/")
}

// absolute makes a page's link absolute with the base URL. The link already
// starts with the base URL's path, so resolving it against the base keeps
// that path once. A site with no base URL configured still builds; its feed
// just carries relative links.
func absolute(base, rel string) string {
	if base == "" {
		return rel
	}
	b, err := url.Parse(base)
	if err != nil {
		return rel
	}
	r, err := url.Parse(rel)
	if err != nil {
		return rel
	}
	return b.ResolveReference(r).String()
}
