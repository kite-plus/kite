package render

import (
	"html/template"
	"maps"
	"reflect"
	"slices"

	"github.com/kite-plus/kite/internal/render/markdown"
)

type shortcodeModel struct {
	call *markdown.Call
	page Page
	site Site
}

// NewShortcode returns the Shortcode view of a call, made by a page's body.
func NewShortcode(call *markdown.Call, page Page, site Site) Shortcode {
	return shortcodeModel{call: call, page: page, site: site}
}

func (s shortcodeModel) Name() string        { return s.call.Name }
func (s shortcodeModel) IsNamedParams() bool { return len(s.call.Named) > 0 }
func (s shortcodeModel) RawInner() string    { return s.call.RawInner }
func (s shortcodeModel) Ordinal() int        { return s.call.Ordinal }
func (s shortcodeModel) Page() Page          { return s.page }
func (s shortcodeModel) Site() Site          { return s.site }

func (s shortcodeModel) Get(key any) any {
	if name, ok := key.(string); ok {
		return s.call.Named[name]
	}
	v := reflect.ValueOf(key)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i := v.Int(); i >= 0 && i < int64(len(s.call.Positional)) {
			return s.call.Positional[i]
		}
	}
	return nil
}

func (s shortcodeModel) Params() any {
	if len(s.call.Named) > 0 {
		return maps.Clone(s.call.Named)
	}
	return slices.Clone(s.call.Positional)
}

func (s shortcodeModel) Inner() (template.HTML, error) {
	html, err := s.call.Inner()
	// Rendered by the markdown pipeline under the site's policy for raw
	// HTML, as a page's content is.
	return template.HTML(html), err //nolint:gosec // rendered by the markdown pipeline
}

// Parent is an untyped nil at the top of a body, so {{ with .Parent }} skips
// it.
func (s shortcodeModel) Parent() Shortcode {
	if s.call.Parent == nil {
		return nil
	}
	return shortcodeModel{call: s.call.Parent, page: s.page, site: s.site}
}

// BodyImage is what a render-image template receives as dot: a picture a
// page's body shows, ![Text](Destination "Title").
type BodyImage interface {
	// Destination is the picture's address as the body writes it, which
	// names a file of the page's bundle as .Resources.Get takes it.
	Destination() string
	// Src is the address a page shows the picture from without a template:
	// under the site's path when the body names it from the site's root.
	Src() string
	Text() string
	Title() string
	Page() Page
	Site() Site
}

type bodyImageModel struct {
	p    markdown.Picture
	page Page
	site Site
}

// NewBodyImage returns the BodyImage view of a picture a page's body shows.
func NewBodyImage(p markdown.Picture, page Page, site Site) BodyImage {
	return bodyImageModel{p: p, page: page, site: site}
}

func (b bodyImageModel) Destination() string { return b.p.Destination }
func (b bodyImageModel) Src() string         { return b.p.Src }
func (b bodyImageModel) Text() string        { return b.p.Text }
func (b bodyImageModel) Title() string       { return b.p.Title }
func (b bodyImageModel) Page() Page          { return b.page }
func (b bodyImageModel) Site() Site          { return b.site }
