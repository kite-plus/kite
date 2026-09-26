package theme

import "github.com/kite-plus/kite/internal/pack"

// PacksDir holds a theme's language packs; see package pack for their shape.
// The rest of a pack, beyond the words under "theme", is left for the words
// a theme's pages say.
const PacksDir = pack.Dir

// Pack returns the words of the pack that best suits a language, or nil when
// the theme speaks no such language and its manifest's own words are all
// there is.
func (t *Theme) Pack(lang string) map[string]string { return t.Packs.Pick(lang) }

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
	say := pack.Say(words)

	m.Title = say("theme.title", m.Title)
	m.Description = say("theme.description", m.Description)
	m.Settings = pack.Schema(m.Settings, "theme.settings.", say)
	m.Layouts = make([]Layout, len(t.Manifest.Layouts))
	for i, l := range t.Manifest.Layouts {
		key := "theme.layouts." + l.Name + "."
		l.Label = say(key+"label", l.Label)
		l.Description = say(key+"description", l.Description)
		m.Layouts[i] = l
	}
	return m
}
