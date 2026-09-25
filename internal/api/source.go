package api

import (
	"context"
	"net/http"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/publish"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/render/url"
)

// View is one consistent look at an open project.
type View struct {
	Reader         content.Reader
	Resolver       *url.Resolver
	Types          *content.Registry
	Site           config.Site
	Build          config.Build
	Store          string
	Runtime        string
	Theme          string
	Version        string
	ConfigRevision content.Revision

	// ActiveTheme is the theme in use, nil for a view that has none. It
	// declares what it can be configured with, so a theme author gets a
	// settings form without writing any admin code.
	ActiveTheme *theme.Theme
	// ThemeSettings is what kite.yaml holds under theme.settings, as written.
	// It is kept raw because each theme reads it through its own schema, and
	// the admin may be looking at a theme other than the one in use.
	ThemeSettings map[string]any
	// Themes lists every theme the project could switch to. It is nil where
	// themes cannot be switched.
	Themes func() []InstalledTheme
	// Previews keeps the sites being drawn with a theme or settings that are
	// being tried. It is nil where none can be drawn.
	Previews Previews

	// Writer is nil when this deployment may not be written to, which is the
	// difference between a preview an author is typing into and a read-only
	// server someone pointed at a repository.
	Writer content.Writer

	// Preview renders an item that is not on disk, through the same renderer,
	// theme and resolver a build uses.
	Preview func(context.Context, *content.Content) ([]byte, error)

	// WordCount counts an item that is not on disk the way its page will.
	WordCount func(context.Context, *content.Content) (int, error)

	// Publisher moves committed content onward. It is nil when the project
	// has none configured, which is a perfectly ordinary way to run: an
	// author may prefer to commit themselves.
	Publisher publish.Publisher

	// Refresh brings the read model and the routing table up to date after a
	// write, and is called before the response is sent.
	//
	// Waiting for the file watcher instead would make a write followed by a
	// read return what the client sent a moment ago, which is exactly the
	// kind of stale answer that teaches a UI to keep its own copy of the
	// truth.
	Refresh func(context.Context) error

	// Problems is the content the index refused, as of this view.
	Problems []string
}

// InstalledTheme is a theme a project could switch to.
type InstalledTheme struct {
	// Name is what theme.name chooses it by.
	Name    string
	Builtin bool
	// Theme is nil when the theme cannot be used. Problem then says why, and
	// Manifest holds what could be read of it, if anything.
	Theme    *theme.Theme
	Manifest *theme.Manifest
	Problem  string
}

// Trial is what a preview draws: a theme, by name, and the settings it is
// given in place of the stored ones.
type Trial struct {
	Theme    string
	Settings map[string]any
}

// Previews keeps sites drawn with a theme or settings being tried, each at an
// address of its own, until it is closed or left unused for long enough.
type Previews interface {
	// Open draws a trial as a site whose address is prefix followed by the
	// token it returns, and every link in it leads there.
	Open(ctx context.Context, trial Trial, prefix string) (token string, err error)
	// Update draws an open preview again with another trial. It reports false
	// for a token that is not open, as when it was left unused and forgotten.
	Update(ctx context.Context, token string, trial Trial) (bool, error)
	// Close forgets a preview.
	Close(token string)
	// Serve answers a request for one of a preview's pages or files, at its
	// full path, reporting false for a token that is not open.
	Serve(w http.ResponseWriter, r *http.Request, token, path string) bool
}

// SiteSource supplies the project's current state.
//
// The API reads through a function rather than holding a project, because a
// serving process reindexes underneath it: which files the index refused
// changes while requests are in flight, and a handler that captured the list
// at startup would go on reporting a problem the author has already fixed.
type SiteSource func() View
