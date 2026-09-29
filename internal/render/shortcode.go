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
