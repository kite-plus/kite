package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"testing/fstest"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/theme"
)

// Bounds on an uploaded theme: the archive as sent, the files once unpacked,
// and how many of them there are. A theme is templates, a stylesheet or two
// and perhaps some fonts and pictures; these leave room for all of that and
// none for an archive built to fill the disk.
const (
	maxThemeArchive = 64 << 20
	maxThemeSize    = 128 << 20
	maxThemeFiles   = 5000
)

// handleThemes lists the themes the project could use.
func (s *Server) handleThemes(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Themes == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot list themes")
		return
	}
	installed := view.Themes()
	out := make([]ThemeInfo, 0, len(installed))
	for _, one := range installed {
		out = append(out, themeInfo(view, installed, one, r))
	}
	writeJSON(w, http.StatusOK, List[ThemeInfo]{Items: out})
}

// handleTheme describes one theme, with its settings as it would read them.
func (s *Server) handleTheme(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	installed, one, ok := s.findTheme(w, view, r.PathValue("name"))
	if !ok {
		return
	}

	detail := ThemeDetail{
		ThemeInfo: themeInfo(view, installed, one, r),
		Values:    map[string]any{},
		Stored:    []string{},
	}
	if th := one.Theme; th != nil {
		m := described(th, r)
		detail.Schema = m.Settings
		detail.Values = th.Manifest.Settings.Resolve(view.ThemeSettings)
		for _, key := range slices.Sorted(maps.Keys(view.ThemeSettings)) {
			if th.Manifest.Settings.Field(key) != nil {
				detail.Stored = append(detail.Stored, key)
			}
		}
		detail.Layouts = layoutsFor(m.Layouts, "")
	}
	// Settings saved from this description go back with this revision.
	w.Header().Set("ETag", etag(view.ConfigRevision))
	writeJSON(w, http.StatusOK, detail)
}

// handleThemeScreenshot serves a theme's picture of itself.
func (s *Server) handleThemeScreenshot(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	_, one, ok := s.findTheme(w, view, r.PathValue("name"))
	if !ok {
		return
	}
	var name string
	if one.Theme != nil {
		name, ok = one.Theme.ScreenshotPath()
	}
	if !ok {
		fail(w, http.StatusNotFound, CodeNotFound, "this theme has no screenshot")
		return
	}
	data, err := fs.ReadFile(one.Theme.Root, name)
	if err != nil {
		s.failErr(w, err)
		return
	}
	sum := sha256.Sum256(data)
	w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:8])+`"`)
	// Asked again each time, since a theme replaced by a new version keeps
	// the address and changes the picture.
	w.Header().Set("Cache-Control", "no-cache")
	if ctype := mime.TypeByExtension(path.Ext(name)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

// handleInstallTheme installs a theme from a zip archive. A theme of the same
// name is replaced only when the request asks for that with replace=true.
func (s *Server) handleInstallTheme(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	if view.Themes == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot install themes")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"send the theme as a zip archive in multipart form data under the name \"file\"")
		return
	}
	defer func() { _ = file.Close() }()
	archive, err := io.ReadAll(io.LimitReader(file, maxThemeArchive+1))
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(archive) > maxThemeArchive {
		fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest,
			fmt.Sprintf("a theme archive may be at most %d MB", maxThemeArchive>>20))
		return
	}

	files, problem := unpackTheme(archive)
	if problem != "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", problem)
		return
	}
	uploaded, err := theme.Load(mapFS(files))
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", err.Error())
		return
	}
	if !uploaded.Manifest.SupportsStatic() {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"the theme says it cannot be built into a static site")
		return
	}

	name := uploaded.Manifest.Name
	if !content.ValidThemeName(name) {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", fmt.Sprintf(
			"the theme is named %q, and a name has to be usable as a directory: letters, digits, dots, - and _", name))
		return
	}
	installed := view.Themes()
	var existing *InstalledTheme
	for i, one := range installed {
		if one.Name != name {
			continue
		}
		if one.Builtin {
			failField(w, http.StatusBadRequest, CodeInvalidRequest, "file", fmt.Sprintf(
				"the name %s belongs to the theme built into Kite; the theme needs a name of its own", name))
			return
		}
		existing = &installed[i]
	}
	if existing != nil && r.URL.Query().Get("replace") != "true" {
		fresh := InstalledTheme{Name: name, Theme: uploaded, Manifest: &uploaded.Manifest}
		writeJSON(w, http.StatusConflict, ThemeExists{
			Error: ErrorDetail{
				Code:    CodeConflict,
				Message: name + " is already installed; send replace=true to replace it",
			},
			Installed: themeInfo(view, installed, *existing, r),
			Uploaded:  themeInfo(view, installed, fresh, r),
		})
		return
	}

	verb := "install"
	if existing != nil {
		verb = "replace"
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops: []content.Op{content.PutTheme{Name: name, Files: files, Replace: existing != nil}},
		Message: strings.TrimSpace(fmt.Sprintf("theme: %s %s %s",
			verb, name, uploaded.Manifest.Version)),
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}

	after := s.src()
	now := after.Themes()
	for _, one := range now {
		if one.Name == name && !one.Builtin {
			writeJSON(w, http.StatusCreated, themeInfo(after, now, one, r))
			return
		}
	}
	s.failErr(w, errNothingWritten)
}

// handleRemoveTheme removes an installed theme. The one built into Kite and
// the one in use stay.
func (s *Server) handleRemoveTheme(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	_, one, ok := s.findTheme(w, view, r.PathValue("name"))
	if !ok {
		return
	}
	switch {
	case one.Builtin:
		fail(w, http.StatusBadRequest, CodeInvalidRequest, "the theme built into Kite cannot be removed")
		return
	case one.Name == view.Theme:
		fail(w, http.StatusConflict, CodeConflict, "the theme in use cannot be removed; switch to another one first")
		return
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.DeleteTheme{Name: one.Name}},
		Message: "theme: remove " + one.Name,
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// findTheme looks a theme up by name, answering for itself when it cannot.
// The theme built into Kite comes first, as it does when a name is chosen.
func (s *Server) findTheme(w http.ResponseWriter, view View, name string) ([]InstalledTheme, InstalledTheme, bool) {
	if view.Themes == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot list themes")
		return nil, InstalledTheme{}, false
	}
	installed := view.Themes()
	for _, one := range installed {
		if one.Name == name {
			return installed, one, true
		}
	}
	fail(w, http.StatusNotFound, CodeNotFound, "no theme of that name is installed: "+name)
	return nil, InstalledTheme{}, false
}

// themeInfo describes a theme in the language a request asks for.
func themeInfo(view View, installed []InstalledTheme, one InstalledTheme, r *http.Request) ThemeInfo {
	info := ThemeInfo{
		Name:    one.Name,
		Title:   one.Name,
		Builtin: one.Builtin,
		Problem: one.Problem,
	}
	// A directory hidden by the built-in theme's name is never the one in
	// use, even though kite.yaml names it.
	info.Active = one.Name == view.Theme && (one.Builtin || !slices.ContainsFunc(installed,
		func(other InstalledTheme) bool { return other.Builtin && other.Name == one.Name }))

	m := one.Manifest
	if one.Theme != nil {
		localized := described(one.Theme, r)
		m = &localized
		if _, ok := one.Theme.ScreenshotPath(); ok {
			info.Screenshot = Prefix + "/themes/" + url.PathEscape(one.Name) + "/screenshot"
		}
	}
	if m == nil {
		return info
	}
	if m.Title != "" {
		info.Title = m.Title
	}
	info.Version = m.Version
	info.Description = m.Description
	info.License = m.License
	info.Homepage = m.Homepage
	info.Tags = m.Tags
	info.Requires = m.Requires
	if m.Author.Name != "" {
		info.Author = &ThemeAuthor{Name: m.Author.Name, URL: m.Author.URL}
	}
	return info
}

// unpackTheme reads a theme out of a zip archive: from its top, or from the
// one folder everything in it sits in, as an archive of a repository has it.
// It reports what is wrong with the archive when it cannot.
func unpackTheme(archive []byte) (map[string][]byte, string) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, "the file is not a zip archive"
	}

	var names []string
	for _, f := range zr.File {
		if name := archived(f.Name); name != "" && !strings.HasSuffix(name, "/") {
			names = append(names, name)
		}
	}
	root := ""
	if !slices.Contains(names, theme.ManifestName) {
		if len(names) == 0 {
			return nil, "the archive is empty"
		}
		top, _, _ := strings.Cut(names[0], "/")
		for _, name := range names {
			if !strings.HasPrefix(name, top+"/") {
				return nil, "the archive has no theme.yaml at its top or in a single folder"
			}
		}
		if !slices.Contains(names, top+"/"+theme.ManifestName) {
			return nil, "the archive has no theme.yaml at its top or in a single folder"
		}
		root = top + "/"
	}

	files := make(map[string][]byte)
	var total int64
	for _, f := range zr.File {
		name := archived(f.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		rel := strings.TrimPrefix(name, root)
		if !fs.ValidPath(rel) {
			return nil, "the archive holds a path that leads out of the theme: " + f.Name
		}
		if !f.Mode().IsRegular() {
			return nil, "the archive holds something other than a plain file: " + f.Name
		}
		if len(files) == maxThemeFiles {
			return nil, fmt.Sprintf("a theme may hold at most %d files", maxThemeFiles)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, "the archive cannot be read: " + err.Error()
		}
		// The sizes an archive declares are not trusted: the bytes are
		// counted as they come out.
		data, err := io.ReadAll(io.LimitReader(rc, maxThemeSize-total+1))
		_ = rc.Close()
		if err != nil {
			return nil, "the archive cannot be read: " + err.Error()
		}
		total += int64(len(data))
		if total > maxThemeSize {
			return nil, fmt.Sprintf("a theme may take at most %d MB unpacked", maxThemeSize>>20)
		}
		files[rel] = data
	}
	return files, ""
}

// archived is the name of a file in an archive with forward slashes, or ""
// for what a computer adds to an archive on its own: a folder of macOS
// metadata, a Finder or Explorer file, a repository's .git.
func archived(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	for part := range strings.SplitSeq(name, "/") {
		switch part {
		case "__MACOSX", ".git", ".DS_Store", "Thumbs.db", "desktop.ini":
			return ""
		}
	}
	return name
}

// mapFS holds unpacked files as a filesystem a theme can be loaded from.
func mapFS(files map[string][]byte) fs.FS {
	fsys := make(fstest.MapFS, len(files))
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: data, Mode: 0o644}
	}
	return fsys
}
