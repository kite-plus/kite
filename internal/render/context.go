// Package render defines the data contract a theme sees.
//
// Everything here is an interface with methods rather than a struct with
// exported fields. Methods can be added without breaking a theme and
// deprecated with a warning; exported fields would freeze Kite's internal
// layout for as long as third-party themes exist.
package render

import (
	"html/template"
	"time"
)

// Kind classifies the page being rendered. It is the first dimension of
// template lookup.
type Kind string

const (
	KindHome     Kind = "home"
	KindSingle   Kind = "single"
	KindList     Kind = "list"
	KindTaxonomy Kind = "taxonomy"
	KindTerm     Kind = "term"
	KindNotFound Kind = "404"
)

// Context is the value a template receives as dot.
type Context interface {
	Site() Site
	Page() Page
	Pages() []Page
	Paginator() Paginator
	Terms() []Term

	// Request is nil during a static build. Themes must reach it only through
	// {{ with .Request }}, which is what keeps a theme working in both
	// runtimes instead of silently requiring a server.
	Request() Request
}

// Site describes the whole site.
type Site interface {
	Title() string
	Description() string
	BaseURL() string
	Language() string
	Params() map[string]any
	ThemeSettings() map[string]any
	Taxonomies() []string

	// BuildTime is the only way a template can learn the time. The clock is
	// frozen for the whole build, because a hidden input such as time.Now
	// would make output uncacheable and incremental builds unsound.
	BuildTime() time.Time

	Version() string
	IsBuild() bool
	IsServe() bool
}

// Page is one renderable item.
type Page interface {
	ID() string
	Kind() Kind
	Type() string
	Title() string
	Slug() string
	Description() string

	Permalink() string
	RelPermalink() string
	Aliases() []string

	Content() template.HTML
	Excerpt() string
	TableOfContents() []Heading
	WordCount() int
	ReadingTime() time.Duration

	Date() time.Time
	PublishDate() time.Time
	Lastmod() time.Time
	Draft() bool

	Params() map[string]any
	Terms(taxonomy string) []Term

	// Prev and Next are the neighbors of a single page among the items of its
	// own kind, by publish date: Prev is the older one, Next the newer. Each
	// is nil at its end of the run, and both are on a page that is listed
	// rather than rendered. Neighbors carry what a listed page carries.
	Prev() Page
	Next() Page
}

// Heading is one table-of-contents entry.
type Heading interface {
	Level() int
	ID() string
	Text() string
}

// Term is one value of a taxonomy.
//
// A term carries an optional page so that a site can give a tag a description
// and a cover image. Deciding this now matters: adding it later would change
// the data every taxonomy template receives.
type Term interface {
	Taxonomy() string
	Name() string
	Slug() string
	Count() int
	Permalink() string
	RelPermalink() string
	Page() Page // nil when the term has no content file
}

// Paginator describes one page of a listing. The URL pattern is configuration;
// this interface is the contract.
type Paginator interface {
	PageNumber() int
	TotalPages() int
	TotalItems() int
	PageSize() int
	HasPrev() bool
	HasNext() bool
	PrevURL() string
	NextURL() string
	FirstURL() string
	LastURL() string
	URL(n int) string
}

// Request exposes the parts of an HTTP request a theme may use. It is nil in a
// static build.
type Request interface {
	Path() string
	Query(key string) string
	Header(key string) string
}
