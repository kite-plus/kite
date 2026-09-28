package theme

import (
	"cmp"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kite-plus/kite/internal/render/markdown"
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
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() == 0
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Pointer, reflect.Interface:
		return rv.IsZero()
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

func (strNS) Upper(s string) string             { return strings.ToUpper(s) }
func (strNS) Lower(s string) string             { return strings.ToLower(s) }
func (strNS) Trim(s string) string              { return strings.TrimSpace(s) }
func (strNS) TrimPrefix(p, s string) string     { return strings.TrimPrefix(s, p) }
func (strNS) TrimSuffix(p, s string) string     { return strings.TrimSuffix(s, p) }
func (strNS) Replace(old, new, s string) string { return strings.ReplaceAll(s, old, new) }
func (strNS) Split(sep, s string) []string      { return strings.Split(s, sep) }
func (strNS) HasPrefix(p, s string) bool        { return strings.HasPrefix(s, p) }
func (strNS) HasSuffix(p, s string) bool        { return strings.HasSuffix(s, p) }
func (strNS) Contains(sub, s string) bool       { return strings.Contains(s, sub) }
func (strNS) Repeat(n int, s string) string     { return strings.Repeat(s, max(0, n)) }
func (strNS) CountWords(s string) int           { return len(strings.Fields(s)) }

// Join joins the items of any list, each in its string form, so the []any
// that front matter lists read as joins like a []string.
func (strNS) Join(sep string, items any) (string, error) {
	s, err := list("str.Join", items)
	if err != nil {
		return "", err
	}
	parts := make([]string, s.Len())
	for i := range parts {
		parts[i] = fmt.Sprint(s.Index(i).Interface())
	}
	return strings.Join(parts, sep), nil
}

// Truncate cuts a string to n runes, appending an ellipsis when it had to cut.
func (strNS) Truncate(n int, s string) string {
	runes := []rune(s)
	if n <= 0 || len(runes) <= n {
		return s
	}
	return strings.TrimSpace(string(runes[:n])) + "…"
}

// collNS holds collection helpers. They take any slice or array, such as
// .Pages or what str.Split returns, and a list they return has the element
// type of the one they were given.
type collNS struct{}

// list reads v as a slice for fn. An array is copied into one, and nil, a
// value that is not set, reads as an empty list.
func list(fn string, v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Invalid:
		return reflect.ValueOf([]any{}), nil
	case reflect.Slice:
		return rv, nil
	case reflect.Array:
		s := reflect.MakeSlice(reflect.SliceOf(rv.Type().Elem()), rv.Len(), rv.Len())
		reflect.Copy(s, rv)
		return s, nil
	default:
		return reflect.Value{}, fmt.Errorf("%s: %T is not a list", fn, v)
	}
}

// Len counts the items of a list or map, or the characters of a string. Not
// set counts as none; anything else cannot be counted.
func (collNS) Len(v any) (int, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Invalid:
		return 0, nil
	case reflect.String:
		return utf8.RuneCountInString(rv.String()), nil
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len(), nil
	default:
		return 0, fmt.Errorf("coll.Len: cannot count %T", v)
	}
}

// First keeps the first n items; a negative n keeps them all.
func (collNS) First(n, items any) (any, error) {
	s, k, err := listAndCount("coll.First", n, items)
	if err != nil {
		return nil, err
	}
	if k < 0 || k > s.Len() {
		k = s.Len()
	}
	return s.Slice(0, k).Interface(), nil
}

// Last keeps the last n items; a negative n keeps them all.
func (collNS) Last(n, items any) (any, error) {
	s, k, err := listAndCount("coll.Last", n, items)
	if err != nil {
		return nil, err
	}
	if k < 0 || k > s.Len() {
		k = s.Len()
	}
	return s.Slice(s.Len()-k, s.Len()).Interface(), nil
}

// After drops the first n items.
func (collNS) After(n, items any) (any, error) {
	s, k, err := listAndCount("coll.After", n, items)
	if err != nil {
		return nil, err
	}
	return s.Slice(min(max(k, 0), s.Len()), s.Len()).Interface(), nil
}

func listAndCount(fn string, n, items any) (reflect.Value, int, error) {
	k, err := whole(fn, n)
	if err != nil {
		return reflect.Value{}, 0, err
	}
	s, err := list(fn, items)
	return s, k, err
}

func (collNS) Reverse(items any) (any, error) {
	s, err := list("coll.Reverse", items)
	if err != nil {
		return nil, err
	}
	out := reflect.MakeSlice(s.Type(), s.Len(), s.Len())
	for i := range s.Len() {
		out.Index(s.Len() - 1 - i).Set(s.Index(i))
	}
	return out.Interface(), nil
}

// In reports whether a list holds needle, comparing string forms so that the
// 1 a template writes finds the "1" a list read from text holds.
func (collNS) In(items, needle any) (bool, error) {
	s, err := list("coll.In", items)
	if err != nil {
		return false, err
	}
	want := fmt.Sprint(needle)
	for i := range s.Len() {
		if fmt.Sprint(s.Index(i).Interface()) == want {
			return true, nil
		}
	}
	return false, nil
}

// Uniq keeps the first of the items that have the same string form.
func (collNS) Uniq(items any) (any, error) {
	s, err := list("coll.Uniq", items)
	if err != nil {
		return nil, err
	}
	out := reflect.MakeSlice(s.Type(), 0, s.Len())
	seen := make(map[string]struct{}, s.Len())
	for i := range s.Len() {
		k := fmt.Sprint(s.Index(i).Interface())
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = reflect.Append(out, s.Index(i))
	}
	return out.Interface(), nil
}

// Sort orders numbers by value, times by time and anything else by its
// string form, which is enough for the tag lists and menus a theme sorts.
func (collNS) Sort(items any) (any, error) {
	s, err := list("coll.Sort", items)
	if err != nil {
		return nil, err
	}
	out := make([]reflect.Value, s.Len())
	for i := range out {
		out[i] = s.Index(i)
	}
	slices.SortStableFunc(out, func(a, b reflect.Value) int { return compare(a.Interface(), b.Interface()) })
	sorted := reflect.MakeSlice(s.Type(), 0, len(out))
	return reflect.Append(sorted, out...).Interface(), nil
}

func compare(a, b any) int {
	if x, err := asNumber("", a); err == nil {
		if y, err := asNumber("", b); err == nil {
			return cmp.Compare(x.f, y.f)
		}
	}
	if x, ok := a.(time.Time); ok {
		if y, ok := b.(time.Time); ok {
			return x.Compare(y)
		}
	}
	return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
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

// dateLayouts are the ways front matter writes a date as text, as Kite reads
// the dates it knows.
var dateLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

// AsTime reads a time, or a date written as text in any of the layouts front
// matter uses, so a theme need not know which layout a value was written in.
func (timeNS) AsTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x, nil
	case string:
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, strings.TrimSpace(x)); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("time.AsTime: %v is not a date", v)
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

// Anchorize turns text into an id usable as a fragment, by the rule headings
// get theirs, so a link a theme builds from a heading's text reaches it.
func (urlNS) Anchorize(s string) string { return markdown.Anchor(strings.TrimSpace(s)) }

// mathNS holds arithmetic helpers. They take numbers of any Go type, and
// integers stay integers: the result is an int when every argument is one,
// so a count kept with math.Add compares with the 8 a template writes, and a
// float64 as soon as any argument is a float.
type mathNS struct{}

// number is a template value read as a number: i holds it when isInt, and f
// holds it either way.
type number struct {
	i     int
	f     float64
	isInt bool
}

func asNumber(fn string, v any) (number, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return number{i: int(rv.Int()), f: float64(rv.Int()), isInt: true}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return number{i: int(rv.Uint()), f: float64(rv.Uint()), isInt: true}, nil
	case reflect.Float32, reflect.Float64:
		return number{f: rv.Float()}, nil
	default:
		return number{}, fmt.Errorf("%s: %v is %T, not a number", fn, v, v)
	}
}

// whole reads a count, which may come from a template that counts in floats
// as long as it has no fraction.
func whole(fn string, v any) (int, error) {
	n, err := asNumber(fn, v)
	switch {
	case err != nil:
		return 0, err
	case n.isInt:
		return n.i, nil
	case n.f != math.Trunc(n.f) || math.IsInf(n.f, 0):
		return 0, fmt.Errorf("%s: %v is not a whole number", fn, v)
	default:
		return int(n.f), nil
	}
}

// operands reads the two arguments of fn; ints reports whether both are
// integers, and so whether to work in integers.
func operands(fn string, a, b any) (x, y number, ints bool, err error) {
	if x, err = asNumber(fn, a); err != nil {
		return x, y, false, err
	}
	if y, err = asNumber(fn, b); err != nil {
		return x, y, false, err
	}
	return x, y, x.isInt && y.isInt, nil
}

func (mathNS) Add(a, b any) (any, error) {
	x, y, ints, err := operands("math.Add", a, b)
	switch {
	case err != nil:
		return nil, err
	case ints:
		return x.i + y.i, nil
	}
	return x.f + y.f, nil
}

func (mathNS) Sub(a, b any) (any, error) {
	x, y, ints, err := operands("math.Sub", a, b)
	switch {
	case err != nil:
		return nil, err
	case ints:
		return x.i - y.i, nil
	}
	return x.f - y.f, nil
}

func (mathNS) Mul(a, b any) (any, error) {
	x, y, ints, err := operands("math.Mul", a, b)
	switch {
	case err != nil:
		return nil, err
	case ints:
		return x.i * y.i, nil
	}
	return x.f * y.f, nil
}

// Div divides two integers as Go does, dropping the remainder; with a float
// on either side the result keeps its fraction.
func (mathNS) Div(a, b any) (any, error) {
	x, y, ints, err := operands("math.Div", a, b)
	switch {
	case err != nil:
		return nil, err
	case y.f == 0:
		return nil, fmt.Errorf("math.Div: division by zero")
	case ints:
		return x.i / y.i, nil
	}
	return x.f / y.f, nil
}

func (mathNS) Mod(a, b any) (int, error) {
	x, err := whole("math.Mod", a)
	if err != nil {
		return 0, err
	}
	y, err := whole("math.Mod", b)
	if err != nil {
		return 0, err
	}
	if y == 0 {
		return 0, fmt.Errorf("math.Mod: division by zero")
	}
	return x % y, nil
}

func (mathNS) Max(a, b any) (any, error) {
	x, y, ints, err := operands("math.Max", a, b)
	switch {
	case err != nil:
		return nil, err
	case ints:
		return max(x.i, y.i), nil
	}
	return math.Max(x.f, y.f), nil
}

func (mathNS) Min(a, b any) (any, error) {
	x, y, ints, err := operands("math.Min", a, b)
	switch {
	case err != nil:
		return nil, err
	case ints:
		return min(x.i, y.i), nil
	}
	return math.Min(x.f, y.f), nil
}

func (mathNS) Ceil(a any) (any, error)  { return rounded("math.Ceil", a, math.Ceil) }
func (mathNS) Floor(a any) (any, error) { return rounded("math.Floor", a, math.Floor) }
func (mathNS) Round(a any) (any, error) { return rounded("math.Round", a, math.Round) }

// rounded leaves an integer as it is, having nothing to round.
func rounded(fn string, a any, round func(float64) float64) (any, error) {
	x, err := asNumber(fn, a)
	switch {
	case err != nil:
		return nil, err
	case x.isInt:
		return x.i, nil
	}
	return round(x.f), nil
}

// Int converts a number or the text of one to an int, dropping any fraction.
// Not set converts to 0.
func (mathNS) Int(v any) (int, error) {
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if i, err := strconv.Atoi(s); err == nil {
			return i, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("math.Int: %q is not a number", v)
		}
		v = f
	}
	if v == nil {
		return 0, nil
	}
	n, err := asNumber("math.Int", v)
	switch {
	case err != nil:
		return 0, err
	case n.isInt:
		return n.i, nil
	case math.IsNaN(n.f) || math.IsInf(n.f, 0):
		return 0, fmt.Errorf("math.Int: %v is not a finite number", v)
	}
	return int(n.f), nil
}

// Float converts a number or the text of one to a float64. Not set converts
// to 0.
func (mathNS) Float(v any) (float64, error) {
	if s, ok := v.(string); ok {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return 0, fmt.Errorf("math.Float: %q is not a number", v)
		}
		return f, nil
	}
	if v == nil {
		return 0, nil
	}
	n, err := asNumber("math.Float", v)
	return n.f, err
}
