package api

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/schema"
)

// Bounds on an uploaded plugin, as for a theme: room for a module, scripts,
// stylesheets and fonts, and none for an archive built to fill the disk.
const (
	maxPluginArchive = 64 << 20
	maxPluginSize    = 128 << 20
	maxPluginFiles   = 5000
)

// PluginAuthor says who made a plugin.
type PluginAuthor struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// PluginInfo describes an installed plugin, in the language the request asks
// for when the plugin has a pack for it.
type PluginInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Version     string        `json:"version,omitempty"`
	Description string        `json:"description,omitempty"`
	Author      *PluginAuthor `json:"author,omitempty"`
	License     string        `json:"license,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Requires    string        `json:"requires,omitempty"`

	// Enabled says the site runs it.
	Enabled bool `json:"enabled"`

	// Injects says it adds code to pages, and Hooks names the build hooks
	// its module runs.
	Injects bool     `json:"injects"`
	Hooks   []string `json:"hooks,omitempty"`

	// Hosts are the other sites its code has visitors' browsers load from,
	// which is worth knowing before turning it on.
	Hosts []string `json:"hosts,omitempty"`

	// Problem says why it cannot be used.
	Problem string `json:"problem,omitempty"`
}

// PluginDetail is a plugin with its settings: the form, what each field
// holds now, and which of them the site stores rather than defaults.
type PluginDetail struct {
	PluginInfo
	Schema schema.Schema  `json:"schema,omitempty"`
	Values map[string]any `json:"values"`
	Stored []string       `json:"stored"`
}

// PluginExists answers an upload of a plugin that is installed already, with
// both versions, so that replacing one is a decision rather than a surprise.
type PluginExists struct {
	Error     ErrorDetail `json:"error"`
	Installed PluginInfo  `json:"installed"`
	Uploaded  PluginInfo  `json:"uploaded"`
}

// handlePlugins lists the plugins the project has.
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.InstalledPlugins == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot list plugins")
		return
	}
	installed := view.InstalledPlugins()
	out := make([]PluginInfo, 0, len(installed))
	for _, one := range installed {
		out = append(out, pluginInfo(view, one, r))
	}
	writeJSON(w, http.StatusOK, List[PluginInfo]{Items: out})
}

// handlePlugin describes one plugin with its settings.
func (s *Server) handlePlugin(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	one, ok := s.findPlugin(w, view, r.PathValue("id"))
	if !ok {
		return
	}
	detail := PluginDetail{PluginInfo: pluginInfo(view, one, r), Values: map[string]any{}, Stored: []string{}}
	if p := one.Plugin; p != nil {
		stored := view.Plugins.Settings[one.ID]
		detail.Schema = p.Localized(language(r)).Settings
		detail.Values = p.Settings(stored)
		for _, key := range slices.Sorted(maps.Keys(stored)) {
			if p.Manifest.Settings.Field(key) != nil {
				detail.Stored = append(detail.Stored, key)
			}
		}
	}
	// Settings saved from this description go back with this revision.
	w.Header().Set("ETag", etag(view.ConfigRevision))
	writeJSON(w, http.StatusOK, detail)
}

// handleInstallPlugin installs a plugin from a zip archive. One already
// installed is replaced only when the request asks with replace=true.
func (s *Server) handleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	if view.InstalledPlugins == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot install plugins")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"send the plugin as a zip archive in multipart form data under the name \"file\"")
		return
	}
	defer func() { _ = file.Close() }()
	archive, err := io.ReadAll(io.LimitReader(file, maxPluginArchive+1))
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(archive) > maxPluginArchive {
		fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest,
			fmt.Sprintf("a plugin archive may be at most %d MB", maxPluginArchive>>20))
		return
	}

	files, problem := unpackArchive(archive, "plugin", plugin.ManifestName, maxPluginSize, maxPluginFiles)
	if problem != "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", problem)
		return
	}
	manifest, err := plugin.ReadManifest(mapFS(files))
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", err.Error())
		return
	}
	// Loaded under the id it gives itself, which is the directory it will
	// be installed in; everything else about it is checked the same way a
	// site checks it when it loads.
	uploaded, err := plugin.Load(mapFS(files), manifest.ID)
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", err.Error())
		return
	}
	id := uploaded.Manifest.ID

	var existing *InstalledPlugin
	for _, one := range view.InstalledPlugins() {
		if one.ID == id {
			existing = &one
		}
	}
	fresh := InstalledPlugin{ID: id, Plugin: uploaded, Manifest: &uploaded.Manifest}
	if existing != nil && r.URL.Query().Get("replace") != "true" {
		writeJSON(w, http.StatusConflict, PluginExists{
			Error: ErrorDetail{
				Code:    CodeConflict,
				Message: id + " is already installed; send replace=true to replace it",
			},
			Installed: pluginInfo(view, *existing, r),
			Uploaded:  pluginInfo(view, fresh, r),
		})
		return
	}

	verb := "install"
	if existing != nil {
		verb = "replace"
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutPlugin{ID: id, Files: files, Replace: existing != nil}},
		Message: fmt.Sprintf("plugin: %s %s %s", verb, id, uploaded.Manifest.Version),
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}

	after := s.src()
	for _, one := range after.InstalledPlugins() {
		if one.ID == id {
			writeJSON(w, http.StatusCreated, pluginInfo(after, one, r))
			return
		}
	}
	s.failErr(w, errNothingWritten)
}

// handleRemovePlugin removes an installed plugin. One the site runs stays:
// turning it off first is what makes removing it a decision about a plugin
// the site already does without.
func (s *Server) handleRemovePlugin(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	one, ok := s.findPlugin(w, view, r.PathValue("id"))
	if !ok {
		return
	}
	if slices.Contains(view.Plugins.Enabled, one.ID) {
		fail(w, http.StatusConflict, CodeConflict, "the plugin is turned on; turn it off before removing it")
		return
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.DeletePlugin{ID: one.ID}},
		Message: "plugin: remove " + one.ID,
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// findPlugin looks a plugin up by id, answering for itself when it cannot.
func (s *Server) findPlugin(w http.ResponseWriter, view View, id string) (InstalledPlugin, bool) {
	if view.InstalledPlugins == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot list plugins")
		return InstalledPlugin{}, false
	}
	for _, one := range view.InstalledPlugins() {
		if one.ID == id {
			return one, true
		}
	}
	fail(w, http.StatusNotFound, CodeNotFound, "no plugin of that id is installed: "+id)
	return InstalledPlugin{}, false
}

// pluginInfo describes a plugin in the language a request asks for.
func pluginInfo(view View, one InstalledPlugin, r *http.Request) PluginInfo {
	info := PluginInfo{
		ID:      one.ID,
		Name:    one.ID,
		Enabled: slices.Contains(view.Plugins.Enabled, one.ID),
		Problem: one.Problem,
	}
	m := one.Manifest
	if one.Plugin != nil {
		localized := one.Plugin.Localized(language(r))
		m = &localized
		info.Hosts = one.Plugin.Hosts()
	}
	if m == nil {
		return info
	}
	if m.Name != "" {
		info.Name = m.Name
	}
	info.Version = m.Version
	info.Description = m.Description
	info.License = m.License
	info.Homepage = m.Homepage
	info.Requires = m.Requires
	info.Injects = len(m.Inject) > 0
	info.Hooks = m.Hooks
	if m.Author.Name != "" {
		info.Author = &PluginAuthor{Name: m.Author.Name, URL: m.Author.URL}
	}
	return info
}

// checkPluginsEnabled reports why a list cannot be the plugins a site runs:
// every one has to be installed and able to load, since turning on a plugin
// that cannot would stop the next build.
func checkPluginsEnabled(view View, value any) string {
	list, ok := value.([]any)
	if !ok && value != nil {
		return "want a list of plugin ids"
	}
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		id, ok := item.(string)
		switch {
		case !ok:
			return "want a list of plugin ids"
		case seen[id]:
			return "lists " + id + " twice"
		}
		seen[id] = true
	}
	if view.InstalledPlugins == nil {
		return ""
	}
	installed := view.InstalledPlugins()
	for _, item := range list {
		id := item.(string)
		i := slices.IndexFunc(installed, func(one InstalledPlugin) bool { return one.ID == id })
		switch {
		case i < 0:
			return "no plugin of that id is installed: " + id
		case installed[i].Plugin == nil:
			return id + " cannot be used: " + installed[i].Problem
		}
	}
	return ""
}

// checkPluginSetting reports why a value cannot be stored under a plugin
// setting, given as the path after plugins.settings., which starts with the
// plugin's id. Removing one, which puts the default back, is always allowed.
func checkPluginSetting(view View, path string, value any) string {
	id, key, ok := strings.Cut(path, ".")
	if !ok || key == "" {
		return "name a setting as plugins.settings.<id>.<key>"
	}
	if value == nil {
		return ""
	}
	if view.InstalledPlugins == nil {
		return "there is no plugin to check this setting against"
	}
	for _, one := range view.InstalledPlugins() {
		if one.ID != id {
			continue
		}
		if one.Plugin == nil {
			return id + " cannot be used: " + one.Problem
		}
		field := one.Plugin.Manifest.Settings.Lookup(strings.Split(key, "."))
		if field == nil {
			return "the plugin " + id + " declares no such setting"
		}
		return field.Check(value)
	}
	return "no plugin of that id is installed: " + id
}
