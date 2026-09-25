package theme

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/schema"
)

// PacksDir holds a theme's language packs, one file per language named after
// it, such as i18n/zh-CN.yaml.
//
// The words that describe the theme in the admin sit under the key "theme":
//
//	theme:
//	  title: ...
//	  description: ...
//	  settings:
//	    accent:
//	      label: ...
//	      help: ...
//	      placeholder: ...
//	      options: {auto: ...}
//	    social:
//	      fields:
//	        github: {label: ...}
//	  layouts:
//	    links: {label: ..., description: ...}
//
// A section's fields are keyed like the section itself, since they are stored
// on its level; the fields of a group or a repeat sit under its "fields". The
// rest of a pack is left for the words a theme's pages say.
const PacksDir = "i18n"

var packName = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// loadPacks reads every language pack a theme has.
func loadPacks(fsys fs.FS, theme string) (map[string]map[string]string, error) {
	entries, err := fs.ReadDir(fsys, PacksDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("theme %s: %w", theme, err)
	}

	packs := make(map[string]map[string]string)
	for _, entry := range entries {
		ext := path.Ext(entry.Name())
		if entry.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		lang := strings.TrimSuffix(entry.Name(), ext)
		where := path.Join(PacksDir, entry.Name())
		if !packName.MatchString(lang) {
			return nil, fmt.Errorf("theme %s: %s is not named after a language, as zh-CN.yaml is", theme, where)
		}
		data, err := fs.ReadFile(fsys, where)
		if err != nil {
			return nil, fmt.Errorf("theme %s: %w", theme, err)
		}
		var tree map[string]any
		if err := yaml.Unmarshal(data, &tree); err != nil {
			return nil, fmt.Errorf("theme %s: %s: %w", theme, where, err)
		}
		words := make(map[string]string)
		flatten("", tree, words)
		packs[strings.ToLower(lang)] = words
	}
	return packs, nil
}

// flatten turns nested keys into dotted ones.
func flatten(prefix string, tree map[string]any, into map[string]string) {
	for key, value := range tree {
		switch v := value.(type) {
		case map[string]any:
			flatten(prefix+key+".", v, into)
		case string:
			into[prefix+key] = v
		case bool, int, int64, uint64, float64:
			into[prefix+key] = fmt.Sprint(v)
		}
	}
}

// Pack returns the words of the pack that best suits a language: the one
// named after it, then the one for its language alone, as zh for zh-CN, then
// any other for the same language. It returns nil when the theme speaks no
// such language, and its manifest's own words are all there is.
func (t *Theme) Pack(lang string) map[string]string {
	if len(t.Packs) == 0 || lang == "" {
		return nil
	}
	lang = strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
	if words, ok := t.Packs[lang]; ok {
		return words
	}
	base, _, _ := strings.Cut(lang, "-")
	if words, ok := t.Packs[base]; ok {
		return words
	}
	names := make([]string, 0, len(t.Packs))
	for name := range t.Packs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if strings.HasPrefix(name, base+"-") {
			return t.Packs[name]
		}
	}
	return nil
}

// Localized returns the manifest in a language, from the theme's pack for it:
// the title and description, the label, help, placeholder and option labels
// of each setting, and the label and description of each layout. A string the
// pack does not have stays as the manifest wrote it.
func (t *Theme) Localized(lang string) Manifest {
	m := t.Manifest
	words := t.Pack(lang)
	if len(words) == 0 {
		return m
	}
	say := func(key, fallback string) string {
		if word := words[key]; word != "" {
			return word
		}
		return fallback
	}

	m.Title = say("theme.title", m.Title)
	m.Description = say("theme.description", m.Description)
	m.Settings = localize(m.Settings, "theme.settings.", say)
	m.Layouts = make([]Layout, len(t.Manifest.Layouts))
	for i, l := range t.Manifest.Layouts {
		key := "theme.layouts." + l.Name + "."
		l.Label = say(key+"label", l.Label)
		l.Description = say(key+"description", l.Description)
		m.Layouts[i] = l
	}
	return m
}

func localize(s schema.Schema, prefix string, say func(key, fallback string) string) schema.Schema {
	if s == nil {
		return nil
	}
	out := make(schema.Schema, len(s))
	for i, f := range s {
		key := prefix + f.Key + "."
		f.Label = say(key+"label", f.Label)
		f.Help = say(key+"help", f.Help)
		f.Placeholder = say(key+"placeholder", f.Placeholder)
		if len(f.Options) > 0 {
			options := make([]schema.Option, len(f.Options))
			for j, o := range f.Options {
				o.Label = say(key+"options."+o.Value, o.Label)
				options[j] = o
			}
			f.Options = options
		}
		switch f.Type {
		case schema.TypeSection:
			f.Fields = localize(f.Fields, prefix, say)
		case schema.TypeGroup, schema.TypeRepeat:
			f.Fields = localize(f.Fields, key+"fields.", say)
		}
		out[i] = f
	}
	return out
}
