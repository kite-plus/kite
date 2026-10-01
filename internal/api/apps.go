package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/apps"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/lock"
)

// AppInfo is a theme or a plugin the index lists, and how the site stands
// with it.
type AppInfo struct {
	Kind        string        `json:"kind"`
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Official    bool          `json:"official"`
	Author      *PluginAuthor `json:"author,omitempty"`
	License     string        `json:"license,omitempty"`
	Homepage    string        `json:"homepage,omitempty"`
	Repo        string        `json:"repo"`
	Tags        []string      `json:"tags,omitempty"`
	// Screenshot is where this server serves a theme's picture of itself,
	// fetched from the index, when it has one.
	Screenshot string `json:"screenshot,omitempty"`
	// Icon is where this server serves the package's icon, when the index
	// names one.
	Icon string `json:"icon,omitempty"`

	// Version is the newest version that works with this Kite, and Problem
	// says why there is none.
	Version string `json:"version,omitempty"`
	Problem string `json:"problem,omitempty"`
	// Loads, Inject and Hooks are what that version does to a site: the
	// other sites it has a reader's browser load from, and for a plugin the
	// pieces of code it puts on pages and the hooks it runs.
	Loads  []string `json:"loads"`
	Inject int      `json:"inject"`
	Hooks  []string `json:"hooks"`

	// Installed is the version the site has, when it came from the index or
	// is the project the index names; Update is a newer one that works with
	// this Kite, and Yanked says the one installed is not to be installed
	// any more.
	Installed string `json:"installed,omitempty"`
	Update    string `json:"update,omitempty"`
	Yanked    bool   `json:"yanked,omitempty"`
	// Delisted says why a package the site has is no longer listed.
	Delisted string `json:"delisted,omitempty"`
	// Other says the site has a theme or a plugin of this name that is not
	// this package, which installing it would replace.
	Other bool `json:"other,omitempty"`
}

// AppVersion is one version of a package the index lists.
type AppVersion struct {
	Version   string   `json:"version"`
	Published string   `json:"published,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	Requires  string   `json:"requires,omitempty"`
	Loads     []string `json:"loads"`
	Inject    int      `json:"inject"`
	Hooks     []string `json:"hooks"`
	Yanked    bool     `json:"yanked,omitempty"`
	// Problem says why it cannot be installed on this Kite.
	Problem string `json:"problem,omitempty"`
}

// AppDetail is a package with every version the index lists, newest first.
type AppDetail struct {
	AppInfo
	Versions []AppVersion `json:"versions"`
}

// AppList is what the index lists, and when the index was fetched.
type AppList struct {
	Items   []AppInfo `json:"items"`
	Fetched string    `json:"fetched"`
	// Offline says the index could not be fetched now, so this is the copy
	// fetched then.
	Offline bool `json:"offline,omitempty"`
}

// AppInstall asks for a version of a package, or for the newest that works
// with this Kite.
type AppInstall struct {
	Version string `json:"version,omitempty"`
}

// AppUpdate asks to update a package, with what its owner agreed to.
type AppUpdate struct {
	Version string     `json:"version,omitempty"`
	Confirm AppConfirm `json:"confirm"`
}

// AppConfirm is what a site's owner agreed an update may do.
type AppConfirm struct {
	// Overwrite agrees to replace files changed since the package was
	// installed.
	Overwrite bool `json:"overwrite,omitempty"`
	// Grant agrees to what a plugin's new version does beyond what it did.
	Grant bool `json:"grant,omitempty"`
}

// UpdatePlan says what updating a package would do.
type UpdatePlan struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Notes string `json:"notes,omitempty"`
	// Changed says the installed files differ from what was installed, which
	// the update would replace; Why says how, for a person to read.
	Changed bool   `json:"changed"`
	Why     string `json:"why,omitempty"`
	// Loads, Inject and Hooks are what the new version does to a site.
	Loads  []string `json:"loads"`
	Inject int      `json:"inject"`
	Hooks  []string `json:"hooks"`
	// For a plugin, what it was agreed to do and its new version goes past:
	// the other sites and the hooks it adds, and Injected, the pieces of
	// code agreed to, when it now injects more. Grows sums them up.
	MoreLoads []string `json:"more_loads"`
	MoreHooks []string `json:"more_hooks"`
	Injected  int      `json:"injected"`
	Grows     bool     `json:"grows"`
}

// UpdateNeeds answers an update that needs an agreement it was not given.
type UpdateNeeds struct {
	Error ErrorDetail `json:"error"`
	Plan  UpdatePlan  `json:"plan"`
}

// standing is how a site stands with the index: the packages whose updates
// come from it, and the name of every package it has.
type standing struct {
	tracked map[string]apps.Tracked
	names   map[string]bool
}

func appKey(kind, id string) string { return kind + "/" + id }

// standing reads what the site has against the index.
func (s *Server) standing(view View, ix *apps.Index) (standing, error) {
	st := standing{tracked: map[string]apps.Tracked{}, names: map[string]bool{}}
	lf := &lock.File{}
	if view.Lock != nil {
		var err error
		if lf, err = view.Lock(); err != nil {
			return st, err
		}
	}
	var pkgs []apps.Installed
	if view.Themes != nil {
		for _, one := range view.Themes() {
			if one.Builtin {
				continue
			}
			pkg := apps.Installed{Kind: "theme", Name: one.Name}
			if m := one.Manifest; m != nil {
				pkg.Version, pkg.Homepage = m.Version, m.Homepage
			}
			pkgs = append(pkgs, pkg)
		}
	}
	if view.InstalledPlugins != nil {
		for _, one := range view.InstalledPlugins() {
			pkg := apps.Installed{Kind: "plugin", Name: one.ID}
			if m := one.Manifest; m != nil {
				pkg.Version, pkg.Homepage = m.Version, m.Homepage
			}
			if one.Plugin != nil {
				g := apps.GrantOf(one.Plugin)
				pkg.Grant = &g
			}
			pkgs = append(pkgs, pkg)
		}
	}
	for _, pkg := range pkgs {
		st.names[appKey(pkg.Kind, pkg.Name)] = true
	}
	for _, t := range apps.Track(pkgs, ix, lf, view.Apps.Source()) {
		st.tracked[appKey(t.Kind, t.Name)] = t
	}
	return st, nil
}

func appInfo(a *apps.App, st standing, lang string) AppInfo {
	info := AppInfo{
		Kind:        a.Kind,
		ID:          a.ID,
		Title:       apps.Text(a.Title, lang),
		Description: apps.Text(a.Description, lang),
		Official:    a.Official,
		License:     a.License,
		Homepage:    a.Homepage,
		Repo:        a.Repo,
		Tags:        a.Tags,
		Loads:       []string{},
		Hooks:       []string{},
	}
	if a.Author != nil {
		info.Author = &PluginAuthor{Name: a.Author.Name, URL: a.Author.URL}
	}
	if a.Screenshot != "" {
		info.Screenshot = Prefix + "/apps/" + url.PathEscape(a.Kind) + "/" + url.PathEscape(a.ID) + "/screenshot"
	}
	if a.Icon != "" {
		info.Icon = Prefix + "/apps/" + url.PathEscape(a.Kind) + "/" + url.PathEscape(a.ID) + "/icon"
	}
	if rel, err := a.Pick("", buildinfo.Version); err == nil {
		info.Version = rel.Version
		info.Loads, info.Inject, info.Hooks = nonNilList(rel.Loads), rel.Inject, nonNilList(rel.Hooks)
	} else {
		info.Problem = err.Error()
	}
	if t, ok := st.tracked[appKey(a.Kind, a.ID)]; ok {
		status := t.Status(buildinfo.Version)
		info.Installed, info.Update, info.Yanked, info.Delisted = t.Version, status.Latest, status.Yanked, status.Delisted
	} else if st.names[appKey(a.Kind, a.ID)] {
		info.Other = true
	}
	return info
}

func nonNilList(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// index reads the index for a request, answering for itself when it cannot.
func (s *Server) index(w http.ResponseWriter, r *http.Request, view View, refresh bool) (*apps.Index, apps.Fetched, bool) {
	if view.Apps == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot install from an index")
		return nil, apps.Fetched{}, false
	}
	ix, got, err := view.Apps.Index(r.Context(), refresh)
	if err != nil {
		s.failApps(w, err)
		return nil, apps.Fetched{}, false
	}
	return ix, got, true
}

// failApps answers a failure of the index or of a package from it.
func (s *Server) failApps(w http.ResponseWriter, err error) {
	if errors.Is(err, apps.ErrUntrusted) {
		fail(w, http.StatusBadGateway, CodeIndexUntrusted, err.Error())
		return
	}
	if errors.Is(err, apps.ErrUnreachable) {
		fail(w, http.StatusBadGateway, CodeIndexUnreachable, err.Error())
		return
	}
	fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
}

// findApp finds the package a request names, answering for itself when the
// index does not list it.
func (s *Server) findApp(w http.ResponseWriter, r *http.Request, ix *apps.Index) (*apps.App, bool) {
	kind, id := r.PathValue("kind"), r.PathValue("id")
	if kind != "theme" && kind != "plugin" {
		fail(w, http.StatusNotFound, CodeNotFound, "the index lists themes and plugins, not "+kind)
		return nil, false
	}
	a := ix.Find(kind, id)
	if a == nil {
		fail(w, http.StatusNotFound, CodeNotFound, fmt.Sprintf("the index has no %s named %s", kind, id))
		return nil, false
	}
	return a, true
}

// handleApps lists the themes and plugins in the index, as the site stands
// with each. A package no longer listed is left out, unless the site has it.
func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind != "" && kind != "theme" && kind != "plugin" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "kind", "kind is theme or plugin")
		return
	}
	ix, got, ok := s.index(w, r, view, q.Get("refresh") == "true")
	if !ok {
		return
	}
	st, err := s.standing(view, ix)
	if err != nil {
		s.failErr(w, err)
		return
	}
	words := strings.Fields(q.Get("q"))
	out := AppList{Items: []AppInfo{}, Fetched: got.At.UTC().Format(time.RFC3339), Offline: got.Offline}
	for _, a := range ix.Apps {
		_, installed := st.tracked[appKey(a.Kind, a.ID)]
		if (kind != "" && a.Kind != kind) || (a.Delisted != "" && !installed) || !a.Matches(words) {
			continue
		}
		out.Items = append(out.Items, appInfo(a, st, language(r)))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleApp describes a package in the index with every version it lists.
func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	a, ok := s.findApp(w, r, ix)
	if !ok {
		return
	}
	st, err := s.standing(view, ix)
	if err != nil {
		s.failErr(w, err)
		return
	}
	detail := AppDetail{AppInfo: appInfo(a, st, language(r)), Versions: []AppVersion{}}
	for _, rel := range a.Versions {
		v := AppVersion{
			Version:   rel.Version,
			Published: rel.Published,
			Notes:     rel.Notes,
			Requires:  rel.Requires,
			Loads:     nonNilList(rel.Loads),
			Inject:    rel.Inject,
			Hooks:     nonNilList(rel.Hooks),
			Yanked:    rel.Yanked,
		}
		if err := a.Fits(rel, buildinfo.Version); err != nil {
			v.Problem = err.Error()
		}
		detail.Versions = append(detail.Versions, v)
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleAppScreenshot serves a theme's picture of itself, which this server
// fetches, so that a browser never has to reach the index's addresses.
func (s *Server) handleAppScreenshot(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	a, ok := s.findApp(w, r, ix)
	if !ok {
		return
	}
	if a.Screenshot == "" {
		fail(w, http.StatusNotFound, CodeNotFound, "this package has no screenshot")
		return
	}
	data, ctype, err := view.Apps.Picture(r.Context(), a.Screenshot)
	if err != nil {
		s.failApps(w, err)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

// handleAppIcon serves a package's icon, which this server fetches as it
// fetches a screenshot. An SVG icon is served sandboxed, so that even opened
// on its own it runs nothing as the studio.
func (s *Server) handleAppIcon(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	a, ok := s.findApp(w, r, ix)
	if !ok {
		return
	}
	if a.Icon == "" {
		fail(w, http.StatusNotFound, CodeNotFound, "this package has no icon")
		return
	}
	data, ctype, err := view.Apps.Icon(r.Context(), a.Icon)
	if err != nil {
		s.failApps(w, err)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

// handleInstallApp installs a package from the index. One of its name that
// the site has already is replaced only when the request asks with
// replace=true.
func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	var req AppInstall
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, CodeInvalidRequest, "the body is not an install request: "+err.Error())
			return
		}
	}
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	if err := ix.Usable(); err != nil {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
		return
	}
	a, ok := s.findApp(w, r, ix)
	if !ok {
		return
	}
	st, err := s.standing(view, ix)
	if err != nil {
		s.failErr(w, err)
		return
	}
	exists := st.names[appKey(a.Kind, a.ID)]
	if exists && r.URL.Query().Get("replace") != "true" {
		fail(w, http.StatusConflict, CodeConflict, fmt.Sprintf(
			"the site has a %s named %s already; send replace=true to replace it", a.Kind, a.ID))
		return
	}
	rel, err := a.Pick(req.Version, buildinfo.Version)
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "version", err.Error())
		return
	}
	pkg, err := view.Apps.Fetch(r.Context(), a, rel)
	if err != nil {
		s.failApps(w, err)
		return
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{pkg.Op(exists)},
		Message: fmt.Sprintf("%s: install %s %s from the index", a.Kind, a.ID, rel.Version),
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	s.answerApp(w, r, http.StatusCreated, a, ix)
}

// answerApp answers with a package as the site now stands with it.
func (s *Server) answerApp(w http.ResponseWriter, r *http.Request, status int, a *apps.App, ix *apps.Index) {
	after := s.src()
	st, err := s.standing(after, ix)
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, status, appInfo(a, st, language(r)))
}

// plan works out what updating a package the site has would do.
func (s *Server) plan(ctx context.Context, view View, t apps.Tracked, version string) (UpdatePlan, *apps.Package, error) {
	rel, err := t.App.Pick(version, buildinfo.Version)
	if err != nil {
		return UpdatePlan{}, nil, err
	}
	next, err := view.Apps.Fetch(ctx, t.App, rel)
	if err != nil {
		return UpdatePlan{}, nil, err
	}
	plan := UpdatePlan{
		Kind: t.Kind, ID: t.Name, From: t.Version, To: rel.Version, Notes: rel.Notes,
		Loads: nonNilList(rel.Loads), Inject: rel.Inject, Hooks: nonNilList(rel.Hooks),
		MoreLoads: []string{}, MoreHooks: []string{},
	}
	if view.PackageTree != nil {
		tree, err := view.PackageTree(t.Kind, t.Name)
		if err != nil {
			return UpdatePlan{}, nil, err
		}
		if plan.Why, err = view.Apps.Changed(ctx, t, tree); err != nil {
			return UpdatePlan{}, nil, err
		}
		plan.Changed = plan.Why != ""
	}
	if next.Origin.Granted != nil {
		g, granted := *next.Origin.Granted, t.Granted()
		plan.Loads, plan.Inject, plan.Hooks, plan.Injected = g.Loads, g.Inject, g.Hooks, granted.Inject
		plan.MoreLoads = added(granted.Loads, g.Loads)
		plan.MoreHooks = added(granted.Hooks, g.Hooks)
		plan.Grows = len(apps.Exceeds(g, granted)) > 0
	}
	return plan, next, nil
}

// added lists what after has that before does not.
func added(before, after []string) []string {
	out := []string{}
	for _, x := range after {
		if !slices.Contains(before, x) {
			out = append(out, x)
		}
	}
	return out
}

// tracked finds the package a request names among those the site has from
// the index, answering for itself when it is not one.
func (s *Server) tracked(w http.ResponseWriter, r *http.Request, view View, ix *apps.Index) (apps.Tracked, bool) {
	a, ok := s.findApp(w, r, ix)
	if !ok {
		return apps.Tracked{}, false
	}
	st, err := s.standing(view, ix)
	if err != nil {
		s.failErr(w, err)
		return apps.Tracked{}, false
	}
	t, ok := st.tracked[appKey(a.Kind, a.ID)]
	if !ok {
		fail(w, http.StatusNotFound, CodeNotFound, fmt.Sprintf(
			"the site has no %s %s from the index to update", a.Kind, a.ID))
		return apps.Tracked{}, false
	}
	return t, true
}

// handleUpdatePlan says what updating a package would do, before it is
// asked to.
func (s *Server) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	t, ok := s.tracked(w, r, view, ix)
	if !ok {
		return
	}
	plan, _, err := s.plan(r.Context(), view, t, r.URL.Query().Get("version"))
	if err != nil {
		s.failApps(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// handleUpdateApp updates a package from the index. One whose files changed
// since it was installed, or a plugin whose new version does more, is
// updated only when the request says that was agreed to.
func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	var req AppUpdate
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, http.StatusBadRequest, CodeInvalidRequest, "the body is not an update request: "+err.Error())
			return
		}
	}
	ix, _, ok := s.index(w, r, view, false)
	if !ok {
		return
	}
	if err := ix.Usable(); err != nil {
		fail(w, http.StatusBadRequest, CodeInvalidRequest, err.Error())
		return
	}
	t, ok := s.tracked(w, r, view, ix)
	if !ok {
		return
	}
	plan, next, err := s.plan(r.Context(), view, t, req.Version)
	if err != nil {
		s.failApps(w, err)
		return
	}
	var needs []string
	if plan.Changed && !req.Confirm.Overwrite {
		needs = append(needs, plan.Why+", and the update would replace that")
	}
	if plan.Grows && !req.Confirm.Grant {
		needs = append(needs, "the new version does more than "+t.Name+" was agreed to do")
	}
	if len(needs) > 0 {
		writeJSON(w, http.StatusConflict, UpdateNeeds{
			Error: ErrorDetail{Code: CodeUpdateNeedsConfirmation, Message: strings.Join(needs, "; ")},
			Plan:  plan,
		})
		return
	}
	if next.Plugin != nil && view.Plugins.Enabled != nil {
		for _, id := range view.Plugins.Enabled {
			if id != t.Name {
				continue
			}
			// An enabled plugin's new version is checked as enabling it
			// is, so that the update never leaves the next build to fail.
			if err := next.Plugin.Check(r.Context(), ""); err != nil {
				fail(w, http.StatusBadRequest, CodeInvalidRequest, t.Name+" "+plan.To+" cannot be used: "+err.Error())
				return
			}
		}
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{next.Op(true)},
		Message: fmt.Sprintf("%s: update %s %s to %s", t.Kind, t.Name, t.Version, plan.To),
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	s.answerApp(w, r, http.StatusOK, t.App, ix)
}
