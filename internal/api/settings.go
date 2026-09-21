package api

import (
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/kite-plus/kite/internal/content"
)

// settable is every configuration path this API will write.
//
// A list rather than a rule, because the configuration file is the one place
// where a mistake breaks the whole site. A client cannot reach a key nobody
// designed a control for, and cannot rewrite the store or the output
// directory through a settings form.
var settable = []string{
	"site.title",
	"site.description",
	"site.baseURL",
	"site.language",
	"theme.name",
	"build.pageSize",
}

// settablePrefix covers the keys a theme declares for itself, which cannot be
// listed here because only the theme knows them.
const settablePrefix = "theme.settings."

// handleSettings describes what can be configured and what it is set to.
func (s *Server) handleSettings(w http.ResponseWriter, _ *http.Request) {
	view := s.src()

	writeJSON(w, http.StatusOK, Settings{
		Site: SiteSettings{
			Title:       view.Site.Title,
			Description: view.Site.Description,
			BaseURL:     view.Site.BaseURL,
			Language:    view.Site.Language,
		},
		Theme: ThemeSettings{
			Name: view.Theme,
			// The schema comes from the theme's own manifest, so a theme
			// author gets a settings form without writing any admin code.
			Schema: view.ThemeSchema,
			Values: view.ThemeValues,
		},
		Writable: append(slices.Clone(settable), settablePrefix+"*"),
	})
}

// handleUpdateSettings writes configuration values.
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	values, ok := decodeJSON[map[string]any](s, w, r)
	if !ok {
		return
	}
	for _, path := range slices.Sorted(maps.Keys(values)) {
		if slices.Contains(settable, path) || strings.HasPrefix(path, settablePrefix) {
			continue
		}
		failField(w, http.StatusBadRequest, CodeInvalidRequest, path,
			"this setting cannot be changed through the API")
		return
	}

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutSettings{Values: values}},
		Message: "settings",
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	s.handleSettings(w, r)
}
