package theme

import (
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/kite-plus/kite/internal/pack"
)

// Words are what a theme's pages say, in a site's language: the words of the
// theme's language packs, and of the site's own packs over them, as
// i18n/zh-CN.yaml writes them:
//
//	read_more: 阅读全文
//	posts:
//	  one: "{{ .Count }} post"
//	  other: "{{ .Count }} posts"
//
// The words under "theme" are what the studio says of the theme.
type Words struct {
	lang    string
	say     map[string]string // in the site's language
	english map[string]string // for a word that language lacks

	parsed sync.Map // a word's text to its template
}

// NewWords picks the words for a language from each set of packs, the first
// set taking precedence, as a site's over its theme's.
func NewWords(lang string, sets ...pack.Packs) *Words {
	pick := func(lang string) map[string]string {
		out := make(map[string]string)
		for i := len(sets) - 1; i >= 0; i-- {
			maps.Copy(out, sets[i].Pick(lang))
		}
		return out
	}
	return &Words{lang: lang, say: pick(lang), english: pick("en")}
}

// T says a word in the site's language: T "read_more", or with what to put
// in it, T "posts" 8 or T "of" (dict "Count" 8 "Total" 13). A number is the
// word's .Count; a map with a Count, or anything else, is what the word
// reads. A count chooses the word's plural form. The word is looked for in
// the site's language, then in English, and is the key when neither has it.
func (w *Words) T(key string, args ...any) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("T %s: takes one value to put in the word, not %d", key, len(args))
	}
	var data any
	var count *float64
	if len(args) == 1 {
		data = args[0]
		if n, err := asNumber("", data); err == nil {
			count, data = &n.f, map[string]any{"Count": data}
		} else if m, ok := data.(map[string]any); ok {
			if n, err := asNumber("", m["Count"]); err == nil {
				count = &n.f
			}
		}
	}
	word, ok := w.find(key, count)
	if !ok {
		return key, nil
	}
	if !strings.Contains(word, "{{") {
		return word, nil
	}
	return w.fill(key, word, data)
}

// Has reports whether there is a word for key, in the site's language or in
// English.
func (w *Words) Has(key string) bool {
	_, ok := w.find(key, nil)
	return ok
}

// Pick returns the words for keys as the packs write them, placeholders and
// all, for a script to say.
func (w *Words) Pick(keys ...string) map[string]string {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if word, ok := w.find(key, nil); ok {
			out[key] = word
		} else {
			out[key] = key
		}
	}
	return out
}

func (w *Words) find(key string, count *float64) (string, bool) {
	if w == nil {
		return "", false
	}
	if word, ok := lookup(w.say, key, count, w.lang); ok {
		return word, true
	}
	return lookup(w.english, key, count, "en")
}

// lookup finds a word in one language: the plural form a count takes there,
// then the word itself, then its "other" form, which every language has.
func lookup(words map[string]string, key string, count *float64, lang string) (string, bool) {
	if count != nil {
		if word, ok := words[key+"."+plural(lang, *count)]; ok {
			return word, true
		}
	}
	if word, ok := words[key]; ok {
		return word, true
	}
	word, ok := words[key+".other"]
	return word, ok
}

// fill puts data into a word that holds placeholders, as {{ .Count }}.
func (w *Words) fill(key, word string, data any) (string, error) {
	var t *texttemplate.Template
	if v, ok := w.parsed.Load(word); ok {
		t = v.(*texttemplate.Template)
	} else {
		parsed, err := texttemplate.New(key).Parse(word)
		if err != nil {
			return "", fmt.Errorf("T %s: %w", key, err)
		}
		w.parsed.Store(word, parsed)
		t = parsed
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("T %s: %w", key, err)
	}
	return b.String(), nil
}

// plural is the CLDR plural category a count takes in a language, for the
// languages whose rules are short enough to keep here; any other takes
// "one" for 1 and "other" for the rest, as English does. A word without the
// form falls back to its "other".
func plural(lang string, n float64) string {
	base, _, _ := strings.Cut(strings.ToLower(lang), "-")
	whole := n == math.Trunc(n) && !math.IsInf(n, 0)
	i := int64(math.Abs(n))
	few := i%10 >= 2 && i%10 <= 4 && (i%100 < 12 || i%100 > 14)
	switch base {
	case "zh", "ja", "ko", "vi", "th", "id", "ms", "lo", "km", "my":
		return "other"
	case "fr", "pt", "hy":
		if i <= 1 {
			return "one"
		}
	case "ru", "uk", "be":
		switch {
		case !whole:
			return "other"
		case i%10 == 1 && i%100 != 11:
			return "one"
		case few:
			return "few"
		}
		return "many"
	case "pl":
		switch {
		case !whole:
			return "other"
		case i == 1:
			return "one"
		case few:
			return "few"
		}
		return "many"
	case "cs", "sk":
		switch {
		case !whole:
			return "many"
		case i == 1:
			return "one"
		case i >= 2 && i <= 4:
			return "few"
		}
	default:
		if whole && i == 1 {
			return "one"
		}
	}
	return "other"
}

// i18nNS holds the words of a site's language.
type i18nNS struct{ words *Words }

// T is the top-level T.
func (n i18nNS) T(key string, args ...any) (string, error) { return n.words.T(key, args...) }

// Has reports whether there is a word for key.
func (n i18nNS) Has(key string) bool { return n.words.Has(key) }

// Words returns the words for keys as a map, which a template writes into a
// script as a JSON object: const words = {{ i18n.Words "copy" "copied" }}.
func (n i18nNS) Words(keys ...string) map[string]string { return n.words.Pick(keys...) }
