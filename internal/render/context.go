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

	// KindAlias is an address an item was once published at, which sends a
	// reader on to where it is now. No template draws it.
	KindAlias Kind = "alias"
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

	// Menus are the site's menus by name, each link in the order the site
	// lists it, as {{ range .Site.Menus.main }}. A menu the site has not
	// written is empty.
	Menus() map[string][]MenuItem

	// Author, Keywords and NoIndex describe the site to search engines; a
	// theme writes them into the head of a page.
	Author() string
	Keywords() []string
	NoIndex() bool

	// HeadHTML and FooterHTML are the site's own code for the end of the head
	// and of the body, such as an analytics snippet, which a theme writes out
	// as it is so that the code survives a change of theme.
	HeadHTML() template.HTML
	FooterHTML() template.HTML

	// BuildTime is the only way a template can learn the time. The clock is
	// frozen for the whole build, because a hidden input such as time.Now
	// would make output uncacheable and incremental builds unsound. Like
	// every date a template sees, it is in the site's time zone when the
	// site names one.
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

	// Images are the pictures the body shows, in order and as its source
	// writes them, for a theme that shows one of them elsewhere: a post's
	// first picture on its card, say, when its front matter names no cover.
	// A theme resolves them as it resolves a cover. A listed page has them
	// too.
	Images() []string

	// PublishDate falls back to Date, when the item was created. A file
	// written by hand may give neither, and then the dates are the zero
	// time, which a theme tests with time.IsZero rather than shows.
	Date() time.Time
	PublishDate() time.Time
	Lastmod() time.Time
	Draft() bool

	Params() map[string]any
	Terms(taxonomy string) []Term

	// Resources are the files of the page's bundle, such as the pictures
	// beside its text; none for a page kept as a single file.
	Resources() ResourceList

	// Prev and Next are the neighbors of a single page among the items of its
	// own kind, by publish date: Prev is the older one, Next the newer. Each
	// is nil at its end of the run, and both are on a page that is listed
	// rather than rendered. Neighbors carry what a listed page carries.
	Prev() Page
	Next() Page
}

// MenuItem is one link of a site's menu.
type MenuItem interface {
	Name() string
	// URL is where the link leads: a path within the site, already under its
	// base path, or a full address as the site wrote it. It is empty for an
	// entry that only heads its children.
	URL() string
	// Params is what the site gives a link beyond its name and address, such
	// as an icon, for the theme to read.
	Params() map[string]any
	Children() []MenuItem
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

// Shortcode is what a shortcode's template receives as dot: one call a body
// makes, as {{< figure src="a.jpg" >}}.
type Shortcode interface {
	Name() string

	// Get is a parameter: by position, as .Get 0, or by name, as
	// .Get "src". It is nil when the call does not give it.
	Get(key any) any
	// Params is every parameter: a list when they are given in order, a map
	// when by name.
	Params() any
	IsNamedParams() bool

	// Inner is what the call encloses, rendered as markdown; RawInner is it
	// as written, for a shortcode that reads it as something else, such as a
	// diagram. Only the words of an inner content that is shown count as the
	// page's words.
	Inner() (template.HTML, error)
	RawInner() string

	// Ordinal counts the calls before this one in the same parent, from 0,
	// and Parent is the call this one is inside, nil at the top of a body.
	Ordinal() int
	Parent() Shortcode

	// Page is the page whose body makes the call. Its body is being drawn,
	// so what is derived from it, such as Content, is empty.
	Page() Page
	Site() Site
}

// Request exposes the parts of an HTTP request a theme may use. It is nil in a
// static build.
type Request interface {
	Path() string
	Query(key string) string
	Header(key string) string
}
