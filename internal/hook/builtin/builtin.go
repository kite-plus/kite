// Package builtin implements Kite's own features as hooks.
//
// These are deliberately not called directly from the build engine. They are
// the first consumers of the hook API, which is how the extension points get
// exercised long before any third party depends on them.
package builtin

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/url"
	"slices"
	"time"

	"github.com/kite-plus/kite/internal/hook"
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
			Base:  hook.Base{HookName: "feed", HookPhase: hook.PhaseBuild, HookVersion: "1"},
			Limit: opts.FeedLimit,
			Kinds: kinds,
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

// Feed writes an RSS 2.0 feed.
type Feed struct {
	hook.Base
	Limit int
	Kinds []string
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate,omitempty"`
	Description string `xml:"description,omitempty"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

// BuildComplete emits the feed.
func (f *Feed) BuildComplete(_ context.Context, b *hook.BuildInfo) error {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}

	channel := rssChannel{
		Title:       b.Site.Title,
		Link:        b.Site.BaseURL,
		Description: b.Site.Description,
		Language:    b.Site.Language,
	}
	for _, p := range b.Pages {
		if p.Item == nil || !p.Indexable {
			continue // listings and taxonomy pages are not feed entries
		}
		if !slices.Contains(f.Kinds, string(p.Item.Kind)) {
			continue
		}
		if len(channel.Items) == limit {
			break
		}
		item := rssItem{
			Title:       p.Title,
			Link:        absolute(b.Site.BaseURL, p.URL),
			GUID:        string(p.Item.ID),
			Description: p.Excerpt,
		}
		if p.Item.PublishedAt != nil {
			item.PubDate = p.Item.PublishedAt.UTC().Format(time.RFC1123Z)
		}
		channel.Items = append(channel.Items, item)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(rssFeed{Version: "2.0", Channel: channel}); err != nil {
		return err
	}
	buf.WriteByte('\n')
	return b.Emit("rss.xml", buf.Bytes())
}

// absolute joins a site-relative URL onto the base URL. A site with no base
// URL configured still builds; its feed just carries relative links.
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
