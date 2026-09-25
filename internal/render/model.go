package render

import (
	"html/template"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/markdown"
	"github.com/kite-plus/kite/internal/render/url"
)

// wordsPerMinute and cjkPerMinute are the reading speeds behind
// Page.ReadingTime. They are conventions, not measurements. CJK characters get
// a pace of their own because each one is counted as a word, and a reader
// covers far more of them in a minute than spaced words.
const (
	wordsPerMinute = 220
	cjkPerMinute   = 400
)

// SiteInfo is the static description of a site, supplied once per build or per
// server start.
type SiteInfo struct {
	Title         string
	Description   string
	BaseURL       string
	Language      string
	Params        map[string]any
	ThemeSettings map[string]any
	Taxonomies    []string
	Version       string

	Author     string
	Keywords   []string
	NoIndex    bool
	HeadHTML   string
	FooterHTML string

	// Location is the zone dates are shown in, nil to show each in the zone
	// it was written with.
	Location *time.Location

	// BuildTime is frozen for the lifetime of a build.
	BuildTime time.Time

	// Build marks a static build as opposed to a live server.
	Build bool
}

type siteModel struct{ info SiteInfo }

// NewSite returns the Site view of a site description.
func NewSite(info SiteInfo) Site { return siteModel{info: info} }

func (s siteModel) Title() string                 { return s.info.Title }
func (s siteModel) Description() string           { return s.info.Description }
func (s siteModel) BaseURL() string               { return s.info.BaseURL }
func (s siteModel) Language() string              { return s.info.Language }
func (s siteModel) Params() map[string]any        { return maps.Clone(s.info.Params) }
func (s siteModel) ThemeSettings() map[string]any { return maps.Clone(s.info.ThemeSettings) }
func (s siteModel) Taxonomies() []string          { return slices.Clone(s.info.Taxonomies) }
func (s siteModel) Author() string                { return s.info.Author }
func (s siteModel) Keywords() []string            { return slices.Clone(s.info.Keywords) }
func (s siteModel) NoIndex() bool                 { return s.info.NoIndex }
func (s siteModel) BuildTime() time.Time          { return in(s.info.BuildTime, s.info.Location) }
func (s siteModel) Version() string               { return s.info.Version }
func (s siteModel) IsBuild() bool                 { return s.info.Build }
func (s siteModel) IsServe() bool                 { return !s.info.Build }

// The site's own code is written by whoever runs it, so it is trusted as the
// theme's templates are.
func (s siteModel) HeadHTML() template.HTML   { return template.HTML(s.info.HeadHTML) }   //nolint:gosec // the site's own markup
func (s siteModel) FooterHTML() template.HTML { return template.HTML(s.info.FooterHTML) } //nolint:gosec // the site's own markup

// in shows a time in a zone, or as it is when there is none.
func in(t time.Time, loc *time.Location) time.Time {
	if loc == nil || t.IsZero() {
		return t
	}
	return t.In(loc)
}

type headingModel struct{ h markdown.Heading }

func (h headingModel) Level() int   { return h.h.Level }
func (h headingModel) ID() string   { return h.h.ID }
func (h headingModel) Text() string { return h.h.Text }

type pageModel struct {
	item       *content.Content
	kind       Kind
	doc        *markdown.Document
	resolver   *url.Resolver
	url        string
	terms      map[string][]Term
	prev, next Page
	loc        *time.Location
}

// PageOptions carries what a page needs beyond the stored item.
type PageOptions struct {
	Kind     Kind
	Rendered *markdown.Document
	Resolver *url.Resolver
	Terms    map[string][]Term

	// Location is the zone the page's dates are shown in, as in SiteInfo.
	Location *time.Location

	// URL is the page's link when it is not an item's own, as for a listing.
	URL string

	// Prev and Next are the neighbors of a single page, nil where there is
	// none.
	Prev, Next Page
}

// NewPage returns the Page view of a stored item.
func NewPage(item *content.Content, opts PageOptions) Page {
	if opts.Kind == "" {
		opts.Kind = KindSingle
	}
	return &pageModel{
		item:     item,
		kind:     opts.Kind,
		doc:      opts.Rendered,
		resolver: opts.Resolver,
		url:      opts.URL,
		terms:    opts.Terms,
		prev:     opts.Prev,
		next:     opts.Next,
		loc:      opts.Location,
	}
}

func (p *pageModel) ID() string        { return string(p.item.ID) }
func (p *pageModel) Kind() Kind        { return p.kind }
func (p *pageModel) Type() string      { return string(p.item.Kind) }
func (p *pageModel) Title() string     { return p.item.Title }
func (p *pageModel) Slug() string      { return p.item.Slug }
func (p *pageModel) Aliases() []string { return slices.Clone(p.item.Aliases) }

func (p *pageModel) Description() string {
	if v, ok := p.item.Meta["description"].(string); ok {
		return v
	}
	return p.Excerpt()
}

func (p *pageModel) RelPermalink() string {
	switch {
	case p.url != "":
		return p.url
	case p.resolver == nil:
		return "/" + p.item.Slug
	}
	return p.resolver.For(p.item)
}

func (p *pageModel) Permalink() string {
	if p.resolver == nil {
		return p.RelPermalink()
	}
	return p.resolver.Absolute(p.RelPermalink())
}

func (p *pageModel) Content() template.HTML {
	if p.doc == nil {
		return ""
	}
	// The pipeline has already escaped or allowed raw HTML according to the
	// site's policy, so this value is trusted by construction.
	return template.HTML(p.doc.HTML) //nolint:gosec // rendered by the markdown pipeline
}

func (p *pageModel) Excerpt() string {
	if p.doc == nil {
		return ""
	}
	return p.doc.Excerpt
}

func (p *pageModel) TableOfContents() []Heading {
	if p.doc == nil {
		return nil
	}
	out := make([]Heading, 0, len(p.doc.TOC))
	for _, h := range p.doc.TOC {
		out = append(out, headingModel{h: h})
	}
	return out
}

func (p *pageModel) WordCount() int {
	if p.doc == nil {
		return 0
	}
	return p.doc.WordCount
}

func (p *pageModel) ReadingTime() time.Duration {
	if p.doc == nil {
		return time.Minute
	}
	cjk := p.doc.CJKCount
	spaced := p.doc.WordCount - cjk

	// Both paces over one denominator, rounded up, so the sum stays in
	// integers.
	const whole = wordsPerMinute * cjkPerMinute
	minutes := max(1, (spaced*cjkPerMinute+cjk*wordsPerMinute+whole-1)/whole)
	return time.Duration(minutes) * time.Minute
}

func (p *pageModel) Date() time.Time { return in(p.item.CreatedAt, p.loc) }

func (p *pageModel) PublishDate() time.Time {
	if p.item.PublishedAt == nil {
		return in(p.item.CreatedAt, p.loc)
	}
	return in(*p.item.PublishedAt, p.loc)
}

func (p *pageModel) Lastmod() time.Time     { return in(p.item.UpdatedAt, p.loc) }
func (p *pageModel) Draft() bool            { return p.item.Status == content.StatusDraft }
func (p *pageModel) Params() map[string]any { return maps.Clone(p.item.Meta) }

func (p *pageModel) Terms(taxonomy string) []Term { return p.terms[taxonomy] }

// The fields are interfaces, so a missing neighbor is an untyped nil and
// {{ with .Page.Next }} skips it.
func (p *pageModel) Prev() Page { return p.prev }
func (p *pageModel) Next() Page { return p.next }

type termModel struct {
	taxonomy string
	name     string
	count    int
	rel      string
	abs      string
	page     Page
}

// NewTerm returns the Term view of one taxonomy value.
func NewTerm(taxonomy, name string, count int, resolver *url.Resolver, locale string, page Page) Term {
	t := termModel{taxonomy: taxonomy, name: name, count: count, page: page}
	if resolver != nil {
		t.rel = resolver.ForTerm(taxonomy, name, locale)
		t.abs = resolver.Absolute(t.rel)
	}
	return t
}

func (t termModel) Taxonomy() string     { return t.taxonomy }
func (t termModel) Name() string         { return t.name }
func (t termModel) Slug() string         { return t.rel }
func (t termModel) Count() int           { return t.count }
func (t termModel) RelPermalink() string { return t.rel }
func (t termModel) Permalink() string    { return t.abs }
func (t termModel) Page() Page           { return t.page }

type paginatorModel struct {
	page, size, total int
	resolver          *url.Resolver
	base              string
}

// NewPaginator returns a paginator over a listing.
func NewPaginator(base string, pageNumber, pageSize, totalItems int, resolver *url.Resolver) Paginator {
	return paginatorModel{
		page:     max(1, pageNumber),
		size:     max(1, pageSize),
		total:    totalItems,
		resolver: resolver,
		base:     base,
	}
}

func (p paginatorModel) PageNumber() int { return p.page }
func (p paginatorModel) PageSize() int   { return p.size }
func (p paginatorModel) TotalItems() int { return p.total }

func (p paginatorModel) TotalPages() int {
	if p.total == 0 {
		return 1
	}
	return (p.total + p.size - 1) / p.size
}

func (p paginatorModel) HasPrev() bool { return p.page > 1 }
func (p paginatorModel) HasNext() bool { return p.page < p.TotalPages() }

func (p paginatorModel) URL(n int) string {
	if p.resolver == nil {
		return p.base
	}
	return p.resolver.ForPage(p.base, n)
}

func (p paginatorModel) PrevURL() string  { return p.URL(max(1, p.page-1)) }
func (p paginatorModel) NextURL() string  { return p.URL(min(p.TotalPages(), p.page+1)) }
func (p paginatorModel) FirstURL() string { return p.URL(1) }
func (p paginatorModel) LastURL() string  { return p.URL(p.TotalPages()) }

type requestModel struct{ r *http.Request }

// NewRequest wraps an HTTP request for template use. It returns nil when there
// is no request, which is what a static build passes.
func NewRequest(r *http.Request) Request {
	if r == nil {
		return nil
	}
	return requestModel{r: r}
}

func (r requestModel) Path() string             { return r.r.URL.Path }
func (r requestModel) Query(key string) string  { return r.r.URL.Query().Get(key) }
func (r requestModel) Header(key string) string { return r.r.Header.Get(key) }

type contextModel struct {
	site      Site
	page      Page
	pages     []Page
	paginator Paginator
	terms     []Term
	request   Request
}

// ContextOptions assembles a render context.
type ContextOptions struct {
	Site      Site
	Page      Page
	Pages     []Page
	Paginator Paginator
	Terms     []Term
	Request   Request
}

// NewContext returns the value a template receives as dot.
func NewContext(opts ContextOptions) Context {
	return &contextModel{
		site:      opts.Site,
		page:      opts.Page,
		pages:     opts.Pages,
		paginator: opts.Paginator,
		terms:     opts.Terms,
		request:   opts.Request,
	}
}

func (c *contextModel) Site() Site           { return c.site }
func (c *contextModel) Page() Page           { return c.page }
func (c *contextModel) Pages() []Page        { return c.pages }
func (c *contextModel) Paginator() Paginator { return c.paginator }
func (c *contextModel) Terms() []Term        { return c.terms }

// Request is nil during a static build; the interface is typed so that
// {{ with .Request }} sees an untyped nil rather than a non-nil interface
// wrapping a nil pointer.
func (c *contextModel) Request() Request { return c.request }
