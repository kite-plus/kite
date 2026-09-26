package api

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/theme"
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
	"site.author",
	"site.keywords",
	"site.timezone",
	"site.noindex",
	"site.headHTML",
	"site.footerHTML",
	"theme.name",
	"build.pageSize",
	"build.feedLimit",
	"plugins.enabled",
}

// Limits on what a form may write. Each is well past any real use, and each
// keeps a pasted mistake from making kite.yaml the size of a book.
const (
	maxKeywords     = 50
	maxKeywordRunes = 50
	maxCodeBytes    = 64 << 10
	maxPageSize     = 100
	maxFeedLimit    = 1000
)

// settablePrefix covers the keys a theme declares for itself, which cannot be
// listed here because only the theme knows them. Each is checked against the
// theme's own schema instead. pluginPrefix does the same for each plugin's.
const (
	settablePrefix = "theme.settings."
	pluginPrefix   = "plugins.settings."
)

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
	case "site.timezone":
		if _, err := (config.Site{Timezone: text}).Location(); err != nil {
			return "not a time zone; try something like Asia/Shanghai or UTC"
		}
	case "site.keywords":
		return checkKeywords(value)
	case "site.noindex":
		if _, ok := value.(bool); value != nil && !ok {
			return "want true or false"
		}
	case "site.headHTML", "site.footerHTML":
		if len(text) > maxCodeBytes {
			return fmt.Sprintf("longer than %d KiB", maxCodeBytes>>10)
		}
	case "build.pageSize":
		return checkCount(value, maxPageSize)
	case "build.feedLimit":
		return checkCount(value, maxFeedLimit)
	}
	// A value of the wrong kind would be written as it came and then fail to
	// load, leaving the project unable to open.
	switch path {
	case "site.description", "site.author", "site.timezone", "site.headHTML", "site.footerHTML":
		if _, ok := value.(string); value != nil && !ok {
			return "want text"
		}
	}
	return ""
}

// checkKeywords reports why a value cannot be the site's keywords, which are
// a list of words or nothing.
func checkKeywords(value any) string {
	if value == nil {
		return ""
	}
	list, ok := value.([]any)
	if !ok {
		return "want a list of words"
	}
	if len(list) > maxKeywords {
		return fmt.Sprintf("at most %d keywords", maxKeywords)
	}
	for _, item := range list {
		word, ok := item.(string)
		if !ok || strings.TrimSpace(word) == "" {
			return "each keyword is a word"
		}
		if utf8.RuneCountInString(word) > maxKeywordRunes {
			return fmt.Sprintf("a keyword is at most %d characters", maxKeywordRunes)
		}
	}
	return ""
}

// checkCount reports why a value cannot be a count from 1 to most. Nothing
// is allowed, and puts the default back.
func checkCount(value any, most int) string {
	if value == nil {
		return ""
	}
	n, ok := value.(float64)
	if !ok || n != math.Trunc(n) || n < 1 || n > float64(most) {
		return fmt.Sprintf("want a whole number from 1 to %d", most)
	}
	return ""
}

// handleSettings describes what can be configured and what it is set to.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	w.Header().Set("ETag", etag(view.ConfigRevision))

	settings := Settings{
		Site: SiteSettings{
			Title:       view.Site.Title,
			Description: view.Site.Description,
			BaseURL:     view.Site.BaseURL,
			Language:    view.Site.Language,
			Author:      view.Site.Author,
			Keywords:    view.Site.Keywords,
			Timezone:    view.Site.Timezone,
			NoIndex:     view.Site.NoIndex,
			HeadHTML:    view.Site.HeadHTML,
			FooterHTML:  view.Site.FooterHTML,
		},
		Build: BuildSettings{
			PageSize:  view.Build.PageSize,
			FeedLimit: view.Build.FeedLimit,
		},
		Theme:    ThemeSettings{Name: view.Theme},
		Writable: append(slices.Clone(settable), settablePrefix+"*"),
	}
	if th := view.ActiveTheme; th != nil {
		// The schema comes from the theme's own manifest, so a theme author
		// gets a settings form without writing any admin code.
		settings.Theme.Schema = described(th, r).Settings
		settings.Theme.Values = th.Manifest.Settings.Resolve(view.ThemeSettings)
	}
	writeJSON(w, http.StatusOK, settings)
}

// usableTheme finds the installed theme a name chooses, or says why there is
// none to switch to.
func usableTheme(view View, value any) (*theme.Theme, string) {
	name, ok := value.(string)
	if !ok || name == "" {
		return nil, "name a theme"
	}
	if view.Themes == nil {
		return nil, "this server cannot switch themes"
	}
	for _, one := range view.Themes() {
		if one.Name != name {
			continue
		}
		if one.Theme == nil {
			return nil, "this theme cannot be used: " + one.Problem
		}
		return one.Theme, ""
	}
	return nil, "no theme of that name is installed"
}

// checkThemeSetting reports why a value cannot be stored under a theme
// setting, given as the path after theme.settings. Removing one, which puts
// the theme's default back, is always allowed.
func checkThemeSetting(th *theme.Theme, path string, value any) string {
	if value == nil {
		return ""
	}
	if th == nil {
		return "there is no theme to check this setting against"
	}
	field := th.Manifest.Settings.Lookup(strings.Split(path, "."))
	if field == nil {
		return "the theme " + th.Manifest.Name + " declares no such setting"
	}
	return field.Check(value)
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
	// A theme's settings are checked against the theme they will belong to:
	// the one this request switches to, if it switches, and otherwise the one
	// in use.
	target := view.ActiveTheme
	if name, switching := values["theme.name"]; switching {
		th, problem := usableTheme(view, name)
		if problem != "" {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, "theme.name", problem)
			return
		}
		target = th
	}
	for _, path := range slices.Sorted(maps.Keys(values)) {
		rest, themed := strings.CutPrefix(path, settablePrefix)
		ofPlugin, plugged := strings.CutPrefix(path, pluginPrefix)
		if !slices.Contains(settable, path) && !themed && !plugged {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, path,
				"this setting cannot be changed through the API")
			return
		}
		// Checked before anything is written, not after. Validating on the
		// reload that follows a write would leave the bad value in the file
		// and the project unable to open.
		problem := checkSetting(path, values[path])
		switch {
		case problem != "":
		case themed:
			problem = checkThemeSetting(target, rest, values[path])
		case plugged:
			problem = checkPluginSetting(view, ofPlugin, values[path])
		case path == "plugins.enabled":
			problem = checkPluginsEnabled(r.Context(), view, values[path])
		}
		if problem != "" {
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
