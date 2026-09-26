// Package pack reads the language packs a theme or a plugin brings, and says
// its settings in a language.
//
// A pack is one file per language in the i18n directory, named after the
// language, such as i18n/zh-CN.yaml. The words that describe the theme or
// plugin in the admin sit under one key, "theme" or "plugin":
//
//	plugin:
//	  title: ...
//	  description: ...
//	  settings:
//	    provider:
//	      label: ...
//	      help: ...
//	      placeholder: ...
//	      options: {giscus: ...}
//	    social:
//	      fields:
//	        github: {label: ...}
//
// A section's fields are keyed like the section itself, since they are stored
// on its level; the fields of a group or a repeat sit under its "fields".
package pack

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

// Dir holds the language packs.
const Dir = "i18n"

var packName = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// Packs holds language packs by lowercased language tag, each a flat table of
// dotted keys, as zh-cn: {plugin.title: ...}.
type Packs map[string]map[string]string

// Load reads every pack in fsys's i18n directory. owner names the theme or
// plugin in errors, as "theme paper".
func Load(fsys fs.FS, owner string) (Packs, error) {
	entries, err := fs.ReadDir(fsys, Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", owner, err)
	}

	packs := make(Packs)
	for _, entry := range entries {
		ext := path.Ext(entry.Name())
		if entry.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		lang := strings.TrimSuffix(entry.Name(), ext)
		where := path.Join(Dir, entry.Name())
		if !packName.MatchString(lang) {
			return nil, fmt.Errorf("%s: %s is not named after a language, as zh-CN.yaml is", owner, where)
		}
		data, err := fs.ReadFile(fsys, where)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", owner, err)
		}
		var tree map[string]any
		if err := yaml.Unmarshal(data, &tree); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", owner, where, err)
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

// Pick returns the words of the pack that best suits a language: the one
// named after it, then the one for its language alone, as zh for zh-CN, then
// any other for the same language. It returns nil when there is no such
// pack, and the manifest's own words are all there is.
func (p Packs) Pick(lang string) map[string]string {
	if len(p) == 0 || lang == "" {
		return nil
	}
	lang = strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
	if words, ok := p[lang]; ok {
		return words
	}
	base, _, _ := strings.Cut(lang, "-")
	if words, ok := p[base]; ok {
		return words
	}
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if strings.HasPrefix(name, base+"-") {
			return p[name]
		}
	}
	return nil
}

// Say looks a key up in words, keeping fallback when the pack lacks it.
func Say(words map[string]string) func(key, fallback string) string {
	return func(key, fallback string) string {
		if word := words[key]; word != "" {
			return word
		}
		return fallback
	}
}

// Schema says a settings schema in the words of a pack: the label, help,
// placeholder and option labels of each field under prefix, such as
// "plugin.settings.".
func Schema(s schema.Schema, prefix string, say func(key, fallback string) string) schema.Schema {
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
			f.Fields = Schema(f.Fields, prefix, say)
		case schema.TypeGroup, schema.TypeRepeat:
			f.Fields = Schema(f.Fields, key+"fields.", say)
		}
		out[i] = f
	}
	return out
}
