package api

import (
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/kite-plus/kite/internal/config"
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

// checkSetting reports why a value cannot be stored, or "" when it can.
//
// The admin offers a list rather than a text box for most of these, but the
// admin is not the only client: the whole point of it going through the
// public API is that something else can too.
func checkSetting(path string, value any) string {
	text, _ := value.(string)

	switch path {
	case "site.language":
		// Empty is allowed and means "use the default"; anything present has
		// to be a language tag, because it becomes the lang attribute on
		// every page and the prefix in every localized URL.
		if text != "" && !config.WellFormedLanguage(text) {
			return "not a language tag; try something like en or zh-CN"
		}
	case "site.baseURL":
		if text == "" {
			return "a site needs an address"
		}
		if u, err := url.Parse(text); err != nil || u.Scheme == "" || u.Host == "" {
			return "not an absolute URL; try something like https://example.com"
		}
	case "site.title":
		if strings.TrimSpace(text) == "" {
			return "a site needs a name"
		}
	}
	return ""
}

// handleSettings describes what can be configured and what it is set to.
func (s *Server) handleSettings(w http.ResponseWriter, _ *http.Request) {
	view := s.src()
	w.Header().Set("ETag", etag(view.ConfigRevision))

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
	revision, ok := s.requireIfMatch(w, r)
	if !ok {
		return
	}

	values, ok := decodeJSON[map[string]any](s, w, r)
	if !ok {
		return
	}
	for _, path := range slices.Sorted(maps.Keys(values)) {
		if !slices.Contains(settable, path) && !strings.HasPrefix(path, settablePrefix) {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, path,
				"this setting cannot be changed through the API")
			return
		}
		// Checked before anything is written, not after. Validating on the
		// reload that follows a write would leave the bad value in the file
		// and the project unable to open.
		if problem := checkSetting(path, values[path]); problem != "" {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, path, problem)
			return
		}
	}

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutSettings{IfRevision: revision, Values: values}},
		Message: "settings",
	}); err != nil {
		if conflict, ok := errors.AsType[*content.ConflictError](err); ok && string(conflict.ID) == "kite.yaml" {
			w.Header().Set("ETag", etag(conflict.Actual))
			fail(w, http.StatusConflict, CodeConflict, "settings changed since they were loaded")
			return
		}
		s.failWrite(w, view, r, err)
		return
	}
	s.handleSettings(w, r)
}
