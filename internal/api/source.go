package api

import (
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/url"
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
