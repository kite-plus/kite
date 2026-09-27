package index

import "github.com/kite-plus/kite/internal/render/markdown"

// summarize reads what a list shows of an item: the best short description
// available for it, how many words its body has and the pictures it shows.
//
// An author-written description always wins: it was written to be read on its
// own, which is more than can be said for the opening of an article. Either
// is markdown, and the summary is plain text, since it is read away from its
// page and ends up in meta tags, where markup would show.
func summarize(description, body string) *markdown.Document {
	doc := markdown.Skim(body)
	if d := markdown.Excerpt(description); d != "" {
		doc.Excerpt = d
	}
	return doc
}
