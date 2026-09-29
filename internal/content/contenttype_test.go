package content_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/content"
)

func declared(kind, dir, route string, taxonomies ...string) *content.Type {
	return &content.Type{Kind: content.Kind(kind), Label: kind, Dir: dir, Route: route,
		Layout: content.LayoutBundle, Taxonomies: taxonomies}
}

// A site declares kinds of its own beside post and page, and declaring them
// again replaces the ones declared before while the built-in ones stay.
func TestASiteDeclaresKindsBesideTheBuiltInOnes(t *testing.T) {
	r := content.DefaultRegistry()
	if err := r.Declare(declared("project", "projects", "/projects/:slug", "tags", "stack")); err != nil {
		t.Fatal(err)
	}
	if got := r.Kinds(); !slices.Equal(got, []content.Kind{"post", "page", "project"}) {
		t.Errorf("kinds = %v", got)
	}
	if got := r.TaxonomyNames(); !slices.Equal(got, []string{"categories", "stack", "tags"}) {
		t.Errorf("taxonomies = %v", got)
	}
	if err := r.Declare(declared("book", "books", "/books/:slug")); err != nil {
		t.Fatal(err)
	}
	if got := r.Kinds(); !slices.Equal(got, []content.Kind{"post", "page", "book"}) || r.Get("project") != nil {
		t.Errorf("kinds after declaring again = %v", got)
	}
	if got := r.FeedKinds(); !slices.Equal(got, []string{"post"}) {
		t.Errorf("feed kinds = %v, want only post", got)
	}
}

// A declared kind that would share a folder, an address or a name with
// another is refused, and nothing it was declared with is kept.
func TestADeclaredKindThatCannotBeUsedIsRefused(t *testing.T) {
	for _, c := range []struct {
		name string
		t    *content.Type
		want string
	}{
		{"built in", declared("post", "notes", "/notes/:slug"), "built in"},
		{"folder", declared("note", "posts", "/notes/:slug"), "content/posts already holds the post items"},
		{"taxonomy", declared("tags", "tag-list", "/t/:slug"), "a taxonomy has the name"},
		{"route", declared("note", "notes", "/notes/"), "hold :slug"},
		{"relative", declared("note", "notes", "notes/:slug"), "start with /"},
		{"kind", declared("Note", "notes", "/notes/:slug"), "lowercase"},
		{"dir", declared("note", "a/b", "/notes/:slug"), "one folder"},
		{"layout", &content.Type{Kind: "note", Dir: "notes", Route: "/n/:slug", Layout: "zip"}, "unknown layout"},
		{"order", &content.Type{Kind: "note", Dir: "notes", Route: "/n/:slug", Layout: content.LayoutBundle, Order: "random"}, "unknown order"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := content.DefaultRegistry()
			ok := declared("book", "books", "/books/:slug")
			err := r.Declare(ok, c.t)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want one containing %q", err, c.want)
			}
			if r.Get("book") != nil {
				t.Error("a kind declared beside a refused one was kept")
			}
		})
	}
	r := content.DefaultRegistry()
	if err := r.Declare(declared("book", "books", "/books/:slug"), declared("book", "reads", "/reads/:slug")); err == nil {
		t.Error("a kind declared twice was accepted")
	}
}
