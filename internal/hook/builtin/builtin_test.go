package builtin_test

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/hook/builtin"
)

// rss is the part of a feed these tests read.
type rss struct {
	Channel struct {
		Self []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"http://www.w3.org/2005/Atom link"`
		Generator *string `xml:"generator"`
		Updated   *string `xml:"lastBuildDate"`
		Items     []struct {
			Title       string   `xml:"title"`
			PubDate     string   `xml:"pubDate"`
			Description string   `xml:"description"`
			Categories  []string `xml:"category"`
			Media       []struct {
				URL    string `xml:"url,attr"`
				Medium string `xml:"medium,attr"`
			} `xml:"http://search.yahoo.com/mrss/ content"`
			GUID struct {
				IsPermaLink string `xml:"isPermaLink,attr"`
				ID          string `xml:",chardata"`
			} `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

// emit runs the completion hooks over pages and returns what they wrote.
func emit(t *testing.T, opts builtin.Options, site hook.SiteInfo, pages ...hook.PageInfo) map[string]string {
	t.Helper()
	bus := hook.NewBus()
	builtin.Register(bus, opts)
	files := make(map[string]string)
	err := bus.BuildComplete(t.Context(), &hook.BuildInfo{
		Site:  site,
		Pages: pages,
		Emit: func(path string, data []byte) error {
			files[path] = string(data)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// feedOf is the feed written for pages.
func feedOf(t *testing.T, opts builtin.Options, site hook.SiteInfo, pages ...hook.PageInfo) rss {
	t.Helper()
	opts.Feed = true
	var feed rss
	if err := xml.Unmarshal([]byte(emit(t, opts, site, pages...)["rss.xml"]), &feed); err != nil {
		t.Fatal(err)
	}
	return feed
}

// post is the page of a post, published at the date given, or undated for "".
func post(id, slug, date string) hook.PageInfo {
	item := &content.Content{ID: content.ID(id), Kind: "post", Slug: slug, Title: slug}
	if date != "" {
		at, err := time.Parse(time.DateOnly, date)
		if err != nil {
			panic(err)
		}
		item.PublishedAt = &at
	}
	return hook.PageInfo{Item: item, URL: "/posts/" + slug + "/", Title: slug, Indexable: true}
}

func titles(feed rss) []string {
	var out []string
	for _, item := range feed.Channel.Items {
		out = append(out, item.Title)
	}
	return out
}

// The feed lists posts as the site's listings do: newest first, a post with
// no date last, and the newer id first between two of one date.
func TestTheFeedIsInTheOrderOfTheSitesListings(t *testing.T) {
	feed := feedOf(t, builtin.Options{FeedKinds: []string{"post"}}, hook.SiteInfo{Title: "Site"},
		post("01J8KQ2P3R4S5T6V7W8X9YZ001", "undated", ""),
		post("01J8KQ2P3R4S5T6V7W8X9YZ002", "older", "2026-01-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ003", "same-day-first", "2026-02-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ004", "same-day-second", "2026-02-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ005", "newest", "2026-03-01"),
	)
	want := []string{"newest", "same-day-second", "same-day-first", "older", "undated"}
	if got := titles(feed); !slices.Equal(got, want) {
		t.Errorf("items = %v, want %v", got, want)
	}

	limited := feedOf(t, builtin.Options{FeedKinds: []string{"post"}, FeedLimit: 2}, hook.SiteInfo{Title: "Site"},
		post("01J8KQ2P3R4S5T6V7W8X9YZ001", "older", "2026-01-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ002", "newest", "2026-03-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ003", "newer", "2026-02-01"),
	)
	if got := titles(limited); !slices.Equal(got, []string{"newest", "newer"}) {
		t.Errorf("with a limit of 2, items = %v, want the two newest", got)
	}
}

// A post's guid is its id, which outlives a change of address. RSS takes a
// guid for the post's address unless it says otherwise, so it says so.
func TestAGuidIsTheIdAndNoAddress(t *testing.T) {
	feed := feedOf(t, builtin.Options{FeedKinds: []string{"post"}}, hook.SiteInfo{Title: "Site", BaseURL: "https://example.com"},
		post("01J8KQ2P3R4S5T6V7W8X9YZ001", "hello", "2026-01-01"))
	if len(feed.Channel.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(feed.Channel.Items))
	}
	guid := feed.Channel.Items[0].GUID
	if guid.ID != "01J8KQ2P3R4S5T6V7W8X9YZ001" || guid.IsPermaLink != "false" {
		t.Errorf("guid = %q, isPermaLink = %q; want the id, and false", guid.ID, guid.IsPermaLink)
	}
}

// The feed names the Kite that wrote it, as the site gives it, and nothing
// when it is given none.
func TestTheFeedNamesItsGenerator(t *testing.T) {
	site := hook.SiteInfo{Title: "Site", BaseURL: "https://example.com"}
	hello := post("01J8KQ2P3R4S5T6V7W8X9YZ001", "hello", "2026-01-01")

	feed := feedOf(t, builtin.Options{Generator: "Kite 0.1.9"}, site, hello)
	if g := feed.Channel.Generator; g == nil || *g != "Kite 0.1.9" {
		t.Errorf("generator = %v, want Kite 0.1.9", g)
	}
	if g := feedOf(t, builtin.Options{}, site, hello).Channel.Generator; g != nil {
		t.Errorf("generator = %q with none given", *g)
	}
}

// Every page names the Kite that built it before </head>, whatever theme drew
// it. A page that names a generator of its own keeps it, and one that does not
// close its head is left as it is.
func TestEveryPageNamesItsGenerator(t *testing.T) {
	transform := func(opts builtin.Options, page string) string {
		t.Helper()
		bus := hook.NewBus()
		builtin.Register(bus, opts)
		doc := hook.HTMLDoc{URL: "/", Kind: "home", HTML: page}
		if err := bus.TransformHTML(t.Context(), &doc); err != nil {
			t.Fatal(err)
		}
		return doc.HTML
	}
	credited := builtin.Options{Generator: "Kite 0.1.9"}

	got := transform(credited, "<html><head><title>x</title></head><body></body></html>")
	if want := `<title>x</title><meta name="generator" content="Kite 0.1.9">` + "\n</head>"; !strings.Contains(got, want) {
		t.Errorf("page = %s\nwant it to hold %s", got, want)
	}
	for _, page := range []string{
		`<html><head><meta name="generator" content="Kite"></head><body></body></html>`,
		`<html><head><META content="A theme" NAME=Generator></head><body></body></html>`,
		`<p>a fragment</p>`,
	} {
		if got := transform(credited, page); got != page {
			t.Errorf("%s became %s", page, got)
		}
	}
	if page := "<html><head></head></html>"; transform(builtin.Options{}, page) != page {
		t.Error("a page names a generator with none given")
	}
}

// A feed gives its own address, for a reader to know where it subscribed:
// under the path of a site published beneath one, and in each copy of the
// feed its own. A site with no address has none to give.
func TestTheFeedGivesItsOwnAddress(t *testing.T) {
	hello := post("01J8KQ2P3R4S5T6V7W8X9YZ001", "hello", "2026-01-01")
	opts := builtin.Options{Feed: true, FeedAliases: []string{"index.xml", "posts/index.xml"}}
	files := emit(t, opts, hook.SiteInfo{Title: "Site", BaseURL: "https://example.github.io/blog"}, hello)

	for name, want := range map[string]string{
		"rss.xml":         "https://example.github.io/blog/rss.xml",
		"index.xml":       "https://example.github.io/blog/index.xml",
		"posts/index.xml": "https://example.github.io/blog/posts/index.xml",
	} {
		var feed rss
		if err := xml.Unmarshal([]byte(files[name]), &feed); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		self := feed.Channel.Self
		if len(self) != 1 || self[0].Href != want || self[0].Rel != "self" || self[0].Type != "application/rss+xml" {
			t.Errorf("%s gives itself as %+v, want %s", name, self, want)
		}
	}

	bare := emit(t, opts, hook.SiteInfo{Title: "Site"}, hello)["rss.xml"]
	if strings.Contains(bare, "atom") {
		t.Errorf("a feed with no address to give names one:\n%s", bare)
	}
}

// The channel is dated by its newest post, which two builds of one site
// agree on, and not by the clock, which they would not. A feed with no dated
// post has no date to give.
func TestTheFeedIsDatedByItsNewestPost(t *testing.T) {
	site := hook.SiteInfo{Title: "Site", BaseURL: "https://example.com"}
	opts := builtin.Options{FeedKinds: []string{"post"}}

	feed := feedOf(t, opts, site,
		post("01J8KQ2P3R4S5T6V7W8X9YZ001", "older", "2026-01-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ002", "newest", "2026-03-01"),
		post("01J8KQ2P3R4S5T6V7W8X9YZ003", "undated", ""))
	if u := feed.Channel.Updated; u == nil || *u != "Sun, 01 Mar 2026 00:00:00 +0000" || *u != feed.Channel.Items[0].PubDate {
		t.Errorf("lastBuildDate = %v, want the newest post's date", u)
	}

	for name, pages := range map[string][]hook.PageInfo{
		"no posts":      nil,
		"undated posts": {post("01J8KQ2P3R4S5T6V7W8X9YZ001", "undated", "")},
	} {
		if u := feedOf(t, opts, site, pages...).Channel.Updated; u != nil {
			t.Errorf("%s: lastBuildDate = %q, want none", name, *u)
		}
	}
}

// A post's categories and tags are its categories in the feed, each once
// however often or however it is written, categories first.
func TestAPostsTermsAreItsCategories(t *testing.T) {
	tagged := post("01J8KQ2P3R4S5T6V7W8X9YZ001", "tagged", "2026-01-01")
	tagged.Item.Taxonomies = map[string][]string{
		"tags":       {"Go", " web dev ", "go", "---", "Notes"},
		"categories": {"Notes"},
	}
	plain := post("01J8KQ2P3R4S5T6V7W8X9YZ002", "plain", "2025-01-01")

	feed := feedOf(t, builtin.Options{FeedKinds: []string{"post"}}, hook.SiteInfo{Title: "Site"}, tagged, plain)
	if got, want := feed.Channel.Items[0].Categories, []string{"Notes", "Go", "web dev"}; !slices.Equal(got, want) {
		t.Errorf("categories = %q, want %q", got, want)
	}
	if got := feed.Channel.Items[1].Categories; len(got) != 0 {
		t.Errorf("a post with no terms has categories %q", got)
	}
}

// A post is described by what its author wrote for it, when there is
// something, and by the opening of its text otherwise.
func TestAPostIsDescribedAsItsAuthorWrote(t *testing.T) {
	var pages []hook.PageInfo
	for i, description := range []any{"  Written for the feed.\n", nil, " ", false} {
		p := post(fmt.Sprintf("01J8KQ2P3R4S5T6V7W8X9YZ00%d", i), fmt.Sprint("post-", i), fmt.Sprintf("2026-01-0%d", 9-i))
		p.Excerpt = "The opening of the text."
		if description != nil {
			p.Item.Meta = map[string]any{"description": description}
		}
		pages = append(pages, p)
	}

	feed := feedOf(t, builtin.Options{FeedKinds: []string{"post"}}, hook.SiteInfo{Title: "Site"}, pages...)
	var got []string
	for _, item := range feed.Channel.Items {
		got = append(got, item.Description)
	}
	want := []string{"Written for the feed.", "The opening of the text.", "The opening of the text.", "The opening of the text."}
	if !slices.Equal(got, want) {
		t.Errorf("descriptions = %q, want %q", got, want)
	}
}

// A post's cover goes with it as Media RSS, at the address its page shows it
// from: a full address as written, one from the site's root under the site's
// path, and any other beside the page. A post with no cover, or with
// cover: false, has none, and a feed with no cover declares no Media RSS.
func TestAPostsCoverGoesWithIt(t *testing.T) {
	site := hook.SiteInfo{Title: "Site", BaseURL: "https://example.github.io/blog"}
	for _, tc := range []struct {
		cover any
		want  string
	}{
		{"cover.jpg", "https://example.github.io/blog/posts/hello/cover.jpg"},
		{"./images/cover.jpg", "https://example.github.io/blog/posts/hello/images/cover.jpg"},
		{"/uploads/cover.jpg", "https://example.github.io/blog/uploads/cover.jpg"},
		{"https://cdn.example.com/c.jpg", "https://cdn.example.com/c.jpg"},
		{"//cdn.example.com/c.jpg", "https://cdn.example.com/c.jpg"},
		{"", ""},
		{false, ""},
		{"data:image/png;base64,iVBORw0KGgo=", ""},
	} {
		hello := post("01J8KQ2P3R4S5T6V7W8X9YZ001", "hello", "2026-01-01")
		hello.URL = "/blog/posts/hello/"
		hello.Item.Meta = map[string]any{"cover": tc.cover}

		raw := emit(t, builtin.Options{Feed: true}, site, hello)["rss.xml"]
		var feed rss
		if err := xml.Unmarshal([]byte(raw), &feed); err != nil {
			t.Fatal(err)
		}
		media := feed.Channel.Items[0].Media
		switch {
		case tc.want == "" && (len(media) != 0 || strings.Contains(raw, "mrss")):
			t.Errorf("cover %#v: the feed gives a picture:\n%s", tc.cover, raw)
		case tc.want != "" && (len(media) != 1 || media[0].URL != tc.want || media[0].Medium != "image"):
			t.Errorf("cover %#v: media = %+v, want an image at %s", tc.cover, media, tc.want)
		}
	}
}
