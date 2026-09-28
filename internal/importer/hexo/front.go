package hexo

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontMatter is a Hexo source's front matter: its keys in order, what each
// holds, and the text a plain value was written as, which is what a date
// without a zone has to be read from.
type frontMatter struct {
	keys   []string
	values map[string]any
	raws   map[string]string
}

// read splits a Hexo source into its front matter and its text.
func read(src string) (frontMatter, string, error) {
	fm := frontMatter{values: map[string]any{}, raws: map[string]string{}}
	data, err := os.ReadFile(src)
	if err != nil {
		return fm, "", err
	}
	text := strings.TrimPrefix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\ufeff")
	front, body, found := split(text)
	if !found {
		return fm, text, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return fm, "", fmt.Errorf("front matter: %w", err)
	}
	if len(doc.Content) == 0 {
		return fm, body, nil
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return fm, "", fmt.Errorf("front matter is not a mapping")
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, node := m.Content[i].Value, m.Content[i+1]
		var v any
		if err := node.Decode(&v); err != nil {
			return fm, "", fmt.Errorf("front matter %s: %w", key, err)
		}
		fm.keys = append(fm.keys, key)
		fm.values[key] = v
		if node.Kind == yaml.ScalarNode {
			fm.raws[key] = strings.TrimSpace(node.Value)
		}
	}
	return fm, body, nil
}

// split finds the front matter at the top of a source, between two lines of
// dashes. Hexo lets the first of them be left out, so a source that opens
// with none has front matter if what comes before its first such line reads
// as keys and values.
func split(text string) (front, body string, found bool) {
	lines := strings.SplitAfter(text, "\n")
	start := 0
	if len(lines) > 0 && isFence(lines[0]) {
		start = 1
	}
	for i := start; i < len(lines); i++ {
		if !isFence(lines[i]) {
			continue
		}
		front, body = strings.Join(lines[start:i], ""), strings.Join(lines[i+1:], "")
		if start == 0 {
			var probe map[string]any
			if err := yaml.Unmarshal([]byte(front), &probe); err != nil || len(probe) == 0 {
				return "", text, false
			}
		}
		return front, body, true
	}
	return "", text, false
}

func isFence(line string) bool {
	line = strings.TrimRight(line, "\n")
	return len(line) >= 3 && strings.Trim(line, "-") == ""
}

func (fm frontMatter) value(key string) any { return fm.values[key] }

// string is a plain value as it was written.
func (fm frontMatter) string(key string) string { return fm.raws[key] }

// raw is a plain value as it was written, and whether there is one.
func (fm frontMatter) raw(key string) (string, bool) {
	v, ok := fm.raws[key]
	return v, ok
}

// strings reads a value that may be one name or a list of them, lists inside
// lists included, as Hexo writes categories, as names in order, each once.
func (fm frontMatter) strings(key string) []string {
	var out []string
	var add func(v any)
	add = func(v any) {
		switch x := v.(type) {
		case nil:
		case []any:
			for _, item := range x {
				add(item)
			}
		default:
			if s := strings.TrimSpace(fmt.Sprint(x)); s != "" && !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	}
	add(fm.values[key])
	return out
}

// consumed are the keys an item's own fields hold, which Kite writes under
// its own names, and must not be written again as params.
var consumed = []string{
	"title", "date", "updated", "published", "permalink",
	"id", "slug", "status", "created_at", "updated_at", "published_at", "deleted_at",
	"aliases", "locale", "lastmod", "draft",
}

// coverKeys are where Hexo themes keep what Kite calls a cover.
var coverKeys = []string{"thumbnail", "index_img", "banner_img", "top_img", "featured_image", "banner"}

// params is what an item of kind keeps of its front matter beside its own
// fields. A layout that only names the kind is left out, an excerpt is the
// description when there is none, and a theme's picture is the cover when
// none is named.
func (fm frontMatter) params(kind string) map[string]any {
	out := map[string]any{}
	for _, k := range fm.keys {
		switch {
		case slices.Contains(consumed, k):
		case kind == "post" && (k == "tags" || k == "categories"):
		case k == "layout" && fm.values[k] == kind:
		default:
			out[k] = fm.values[k]
		}
	}
	if _, ok := out["description"]; !ok {
		if s := fm.string("excerpt"); s != "" {
			out["description"] = s
			delete(out, "excerpt")
		}
	}
	if _, ok := out["cover"]; !ok {
		for _, k := range coverKeys {
			if s := fm.string(k); s != "" {
				out["cover"] = s
				break
			}
		}
	}
	if _, ok := out["cover"]; !ok {
		if photos := fm.strings("photos"); len(photos) > 0 {
			out["cover"] = photos[0]
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
