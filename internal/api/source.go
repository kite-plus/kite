package api

import (
	"context"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/url"
	"github.com/kite-plus/kite/internal/schema"
)

// View is one consistent look at an open project.
type View struct {
	Reader   content.Reader
	Resolver *url.Resolver
	Types    *content.Registry
	Site     config.Site
	Store    string
	Runtime  string
	Theme    string
	Version  string

	// ThemeSchema is what the theme declares it can be configured with, and
	// ThemeValues what it is configured to. A theme author gets a settings
	// form out of the first without writing any admin code.
	ThemeSchema schema.Schema
	ThemeValues map[string]any

	// Writer is nil when this deployment may not be written to, which is the
	// difference between a preview an author is typing into and a read-only
	// server someone pointed at a repository.
	Writer content.Writer

	// Preview renders an item that is not on disk, through the same renderer,
	// theme and resolver a build uses.
	Preview func(context.Context, *content.Content) ([]byte, error)

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

// SiteSource supplies the project's current state.
//
// The API reads through a function rather than holding a project, because a
// serving process reindexes underneath it: which files the index refused
// changes while requests are in flight, and a handler that captured the list
// at startup would go on reporting a problem the author has already fixed.
type SiteSource func() View
