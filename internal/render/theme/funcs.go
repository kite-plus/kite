package theme

import (
	"fmt"
	"html/template"
	"math"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Namespaces exist because a flat function table is a permanent compatibility
// liability: once a theme uses a global named "title", that name can never be
// given a different meaning or a different signature. Go templates reach a
// namespace by chaining off a niladic function, so {{ str.Title .X }} resolves
// to strNS.Title.
//
// Only the handful of helpers that the template language itself needs are
// registered at the top level.
func baseFuncs(links Links) template.FuncMap {
	return template.FuncMap{
		// Namespaces.
		"str":         func() strNS { return strNS{} },
		"collections": func() collNS { return collNS{} },
		"coll":        func() collNS { return collNS{} },
		"time":        func() timeNS { return timeNS{} },
		"url":         func() urlNS { return urlNS{links: links} },
		"math":        func() mathNS { return mathNS{} },

		// Top level, because the template language cannot express them any
		// other way.
		"dict":         dict,
		"slice":        func(items ...any) []any { return items },
		"default":      defaultValue,
		"safeHTML":     func(s string) template.HTML { return template.HTML(s) },         //nolint:gosec // explicit opt-in by the theme
		"safeURL":      func(s string) template.URL { return template.URL(s) },           //nolint:gosec // explicit opt-in by the theme
		"safeCSS":      func(s string) template.CSS { return template.CSS(s) },           //nolint:gosec // explicit opt-in by the theme
		"safeJS":       func(s string) template.JS { return template.JS(s) },             //nolint:gosec // explicit opt-in by the theme
		"safeHTMLAttr": func(s string) template.HTMLAttr { return template.HTMLAttr(s) }, //nolint:gosec // explicit opt-in by the theme
	}
}

// dict builds a map from alternating key and value arguments, which is how a
// template passes several values to a partial.
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: expected an even number of arguments, got %d", len(values))
	}
	out := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key %d is %T, want string", i, values[i])
		}
		out[key] = values[i+1]
	}
	return out, nil
}

func defaultValue(fallback, value any) any {
	if isEmpty(value) {
		return fallback
	}
	return value
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case int:
		return x == 0
	case float64:
		return x == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	default:
		return false
	}
}

// strNS holds string helpers.
type strNS struct{}

// Title upper-cases the first letter of each word, leaving the rest alone so
// that "iOS release" does not become "IOS Release".
func (strNS) Title(s string) string {
	prevIsSeparator := true
	return strings.Map(func(r rune) rune {
		out := r
		if prevIsSeparator {
			out = unicode.ToTitle(r)
		}
		prevIsSeparator = !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\''
		return out
	}, s)
}

func (strNS) Upper(s string) string                  { return strings.ToUpper(s) }
func (strNS) Lower(s string) string                  { return strings.ToLower(s) }
func (strNS) Trim(s string) string                   { return strings.TrimSpace(s) }
func (strNS) TrimPrefix(p, s string) string          { return strings.TrimPrefix(s, p) }
func (strNS) TrimSuffix(p, s string) string          { return strings.TrimSuffix(s, p) }
func (strNS) Replace(old, new, s string) string      { return strings.ReplaceAll(s, old, new) }
func (strNS) Split(sep, s string) []string           { return strings.Split(s, sep) }
func (strNS) Join(sep string, parts []string) string { return strings.Join(parts, sep) }
func (strNS) HasPrefix(p, s string) bool             { return strings.HasPrefix(s, p) }
func (strNS) HasSuffix(p, s string) bool             { return strings.HasSuffix(s, p) }
func (strNS) Contains(sub, s string) bool            { return strings.Contains(s, sub) }
func (strNS) Repeat(n int, s string) string          { return strings.Repeat(s, max(0, n)) }
func (strNS) CountWords(s string) int                { return len(strings.Fields(s)) }

// Truncate cuts a string to n runes, appending an ellipsis when it had to cut.
func (strNS) Truncate(n int, s string) string {
	runes := []rune(s)
	if n <= 0 || len(runes) <= n {
		return s
	}
	return strings.TrimSpace(string(runes[:n])) + "…"
}

// collNS holds collection helpers.
type collNS struct{}

func (collNS) Len(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case []string:
		return len(x)
	case string:
		return len([]rune(x))
	case map[string]any:
		return len(x)
	default:
		return 0
	}
}

func (collNS) First(n int, items []any) []any {
	if n < 0 || n > len(items) {
		n = len(items)
	}
	return items[:n]
}

func (collNS) Last(n int, items []any) []any {
	if n < 0 || n > len(items) {
		n = len(items)
	}
	return items[len(items)-n:]
}

func (collNS) After(n int, items []any) []any {
	if n < 0 {
		n = 0
	}
	if n > len(items) {
		return nil
	}
	return items[n:]
}

func (collNS) Reverse(items []any) []any {
	out := slices.Clone(items)
	slices.Reverse(out)
	return out
}

func (collNS) In(items []any, needle any) bool {
	return slices.ContainsFunc(items, func(v any) bool { return fmt.Sprint(v) == fmt.Sprint(needle) })
}

func (collNS) Uniq(items []any) []any {
	seen := make(map[string]struct{}, len(items))
	var out []any
	for _, v := range items {
		k := fmt.Sprint(v)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Sort orders values by their string form, which is enough for the tag lists
// and menus a theme sorts.
func (collNS) Sort(items []any) []any {
	out := slices.Clone(items)
	sort.SliceStable(out, func(i, j int) bool { return fmt.Sprint(out[i]) < fmt.Sprint(out[j]) })
	return out
}

// timeNS holds time helpers.
//
// There is deliberately no Now: a template that reads the clock is a hidden
// input that makes its output uncacheable. Templates use .Site.BuildTime.
type timeNS struct{}

func (timeNS) Format(layout string, t time.Time) string { return t.Format(layout) }
func (timeNS) Year(t time.Time) int                     { return t.Year() }
func (timeNS) Unix(t time.Time) int64                   { return t.Unix() }
func (timeNS) IsZero(t time.Time) bool                  { return t.IsZero() }

// Minutes rounds a duration up to whole minutes, for reading times.
func (timeNS) Minutes(d time.Duration) int { return int((d + time.Minute - 1) / time.Minute) }

func (timeNS) Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// Links is what the url namespace asks of the site it renders, which is its
// URL resolver.
type Links interface {
	// Named is the URL of a page Kite plans: "home", "list" and a content
	// kind, "taxonomy" and a taxonomy, or "term", a taxonomy and a term.
	Named(name string, args ...string) (string, error)
	// Rel is the link to a path within the site, such as "rss.xml".
	Rel(path string) string
	// Absolute makes a link absolute with the site's base URL.
	Absolute(link string) string
}

// urlNS holds link helpers. Themes must not assemble paths themselves: For
// and Rel ask the site's resolver, which knows the routes and the base path
// of a site published under a directory of its host.
type urlNS struct{ links Links }

// For links to a page Kite plans, by name: {{ url.For "home" }},
// {{ url.For "list" "post" }}, {{ url.For "taxonomy" "tags" }} or
// {{ url.For "term" "tags" "Go" }}.
func (n urlNS) For(name string, args ...string) (string, error) {
	if n.links == nil {
		return "", fmt.Errorf("url.For: there is no site to link within")
	}
	return n.links.Named(name, args...)
}

// Rel links to a path within the site, such as "rss.xml" or a path an author
// wrote in the theme settings. A full URL is left as it is.
func (n urlNS) Rel(p string) string {
	if n.links == nil {
		return p
	}
	return n.links.Rel(p)
}

// Abs is [urlNS.Rel] made absolute with the site's base URL.
func (n urlNS) Abs(p string) string {
	if n.links == nil {
		return p
	}
	return n.links.Absolute(n.links.Rel(p))
}

func (urlNS) Escape(s string) string { return url.PathEscape(s) }
func (urlNS) Query(s string) string  { return url.QueryEscape(s) }

func (urlNS) Join(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.Trim(p, "/"); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return "/" + strings.Join(cleaned, "/")
}

// Anchorize turns arbitrary text into an id usable as a fragment.
func (urlNS) Anchorize(s string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r > 127:
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// mathNS holds arithmetic helpers.
type mathNS struct{}

func (mathNS) Add(a, b float64) float64 { return a + b }
func (mathNS) Sub(a, b float64) float64 { return a - b }
func (mathNS) Mul(a, b float64) float64 { return a * b }
func (mathNS) Max(a, b float64) float64 { return math.Max(a, b) }
func (mathNS) Min(a, b float64) float64 { return math.Min(a, b) }
func (mathNS) Ceil(a float64) float64   { return math.Ceil(a) }
func (mathNS) Floor(a float64) float64  { return math.Floor(a) }
func (mathNS) Round(a float64) float64  { return math.Round(a) }

func (mathNS) Div(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("math.Div: division by zero")
	}
	return a / b, nil
}

func (mathNS) Mod(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("math.Mod: division by zero")
	}
	return a % b, nil
}
