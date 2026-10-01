// Package apps reads the index of themes and plugins that Kite installs by
// name, and fetches their archives.
//
// The index is a static JSON file built by the kite-plus/apps repository
// (docs/design/app-center.md 4.1). Every archive it lists is named by its
// sha256, so an address that serves it need not be trusted: bytes that do
// not match are refused.
package apps

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/render/theme"
)

// Format is the format of the index this Kite reads.
const Format = 1

// Indexes are where Kite reads the index from, in order, unless a site names
// one of its own. The first names the index in kite.lock.
var Indexes = []string{
	"https://cdn.jsdelivr.net/gh/kite-plus/apps@main/index.json",
	"https://raw.githubusercontent.com/kite-plus/apps/main/index.json",
}

// Index is the list of themes and plugins.
type Index struct {
	Format    int    `json:"format"`
	Generated string `json:"generated"`
	Apps      []*App `json:"apps"`
}

// App is a theme or a plugin the index lists.
type App struct {
	Kind        string            `json:"kind"`
	ID          string            `json:"id"`
	Official    bool              `json:"official"`
	Repo        string            `json:"repo"`
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description,omitempty"`
	Author      *Author           `json:"author,omitempty"`
	License     string            `json:"license"`
	Homepage    string            `json:"homepage,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Screenshot  string            `json:"screenshot,omitempty"`
	// Icon is the address of a square picture of the package.
	Icon string `json:"icon,omitempty"`
	// Delisted says why the package is no longer listed. Its versions stay,
	// so that a site that has it can be told.
	Delisted string `json:"delisted,omitempty"`
	// Versions are newest first.
	Versions []*Release `json:"versions"`
}

// Author is who made a package.
type Author struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Release is one version of a package.
type Release struct {
	Version   string `json:"version"`
	Published string `json:"published"`
	Notes     string `json:"notes"`
	// API is the contract the package is written to.
	API      string  `json:"api"`
	Requires string  `json:"requires,omitempty"`
	Archive  Archive `json:"archive"`
	// Loads are the other sites the package has a reader's browser load
	// from; Inject and Hooks are what a plugin puts on pages and runs while
	// a site is built.
	Loads  []string `json:"loads"`
	Inject int      `json:"inject,omitempty"`
	Hooks  []string `json:"hooks,omitempty"`
	Yanked bool     `json:"yanked,omitempty"`
}

// Archive is where a release's zip archive is served, and what it is.
type Archive struct {
	URLs   []string `json:"urls"`
	SHA256 string   `json:"sha256"`
	Size   int64    `json:"size"`
}

// Usable says why this Kite cannot install from the index, or nil. An index
// of a later format can still be read for what it lists.
func (ix *Index) Usable() error {
	if ix.Format != Format {
		return fmt.Errorf("the index is format %d, and this Kite reads format %d; upgrade Kite to install from it",
			ix.Format, Format)
	}
	return nil
}

// Find finds a package by kind, theme or plugin, and id.
func (ix *Index) Find(kind, id string) *App {
	for _, a := range ix.Apps {
		if a.Kind == kind && a.ID == id {
			return a
		}
	}
	return nil
}

// Release finds a version of the package.
func (a *App) Release(version string) *Release {
	version = strings.TrimPrefix(version, "v")
	for _, r := range a.Versions {
		if r.Version == version {
			return r
		}
	}
	return nil
}

// apiVersions are the contracts this Kite reads, by kind of package.
var apiVersions = map[string]string{"theme": theme.APIVersion, "plugin": plugin.APIVersion}

// Fits says why a release cannot run on Kite version kite, or nil. A Kite
// built from source, whose version is no release, is not held to a range.
func (a *App) Fits(r *Release, kite string) error {
	if want := apiVersions[a.Kind]; r.API != want {
		return fmt.Errorf("%s %s %s is written to %s, and this Kite reads %s", a.Kind, a.ID, r.Version, r.API, want)
	}
	return theme.Requires(a.Kind, a.ID, r.Requires, kite)
}

// Pick chooses the release to install on Kite version kite: the version
// asked for, or else the newest that is not yanked and fits.
func (a *App) Pick(want, kite string) (*Release, error) {
	if a.Delisted != "" {
		return nil, fmt.Errorf("%s %s is no longer listed: %s", a.Kind, a.ID, a.Delisted)
	}
	if want != "" {
		r := a.Release(want)
		switch {
		case r == nil:
			return nil, fmt.Errorf("%s %s has no version %s; the index lists %s", a.Kind, a.ID, want, a.versionList())
		case r.Yanked:
			return nil, fmt.Errorf("%s %s %s is yanked and is not installed any more", a.Kind, a.ID, r.Version)
		}
		if err := a.Fits(r, kite); err != nil {
			return nil, err
		}
		return r, nil
	}
	var unfit error
	for _, r := range a.Versions {
		if r.Yanked {
			continue
		}
		err := a.Fits(r, kite)
		if err == nil {
			return r, nil
		}
		if unfit == nil {
			unfit = err
		}
	}
	if unfit != nil {
		return nil, fmt.Errorf("no version of %s %s works with this Kite: %w", a.Kind, a.ID, unfit)
	}
	return nil, errors.New(a.Kind + " " + a.ID + " has no version to install")
}

func (a *App) versionList() string {
	var list []string
	for _, r := range a.Versions {
		list = append(list, r.Version)
	}
	if len(list) == 0 {
		return "none"
	}
	return strings.Join(list, ", ")
}

// Text is a title or a description in a language, or in English where the
// package says nothing in that one.
func Text(words map[string]string, lang string) string {
	if t := words[lang]; t != "" {
		return t
	}
	return words["en"]
}

// Matches reports whether every word is found in what the index says of a
// package, in any language, ignoring case.
func (a *App) Matches(words []string) bool {
	var hay []string
	hay = append(hay, a.ID, a.Kind)
	for _, t := range a.Title {
		hay = append(hay, t)
	}
	for _, t := range a.Description {
		hay = append(hay, t)
	}
	hay = append(hay, a.Tags...)
	text := strings.ToLower(strings.Join(hay, "\n"))
	for _, w := range words {
		if !strings.Contains(text, strings.ToLower(w)) {
			return false
		}
	}
	return true
}

// ProjectOf reports whether a homepage is this package's repository, so that
// a package installed from somewhere else is offered updates only when it is
// the same project, never because another one shares its name.
func (a *App) ProjectOf(homepage string) bool {
	home := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(homepage), "/"))
	home = strings.TrimSuffix(home, ".git")
	return a.Repo != "" && home == "https://github.com/"+strings.ToLower(a.Repo)
}
