package index

import "github.com/kite-plus/kite/internal/render/markdown"

// summarize picks the best short description available for an item.
//
// An author-written description always wins: it was written to be read on its
// own, which is more than can be said for the opening of an article. Either
// is markdown, and the summary is plain text, since it is read away from its
// page and ends up in meta tags, where markup would show.
func summarize(description, body string) string {
	if d := markdown.Excerpt(description); d != "" {
		return d
	}
	return markdown.Excerpt(body)
}
