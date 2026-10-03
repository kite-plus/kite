package builtin_test

import (
	"encoding/xml"
	"slices"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/hook/builtin"
)

// rss is the part of a feed these tests read.
type rss struct {
	Channel struct {
		Items []struct {
			Title string `xml:"title"`
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
