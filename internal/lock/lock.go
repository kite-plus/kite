// Package lock reads and writes kite.lock, which records where each theme and
// plugin installed from an index came from.
//
// The files themselves stay in the site's repository, under themes/ and
// plugins/, so that a build never needs the network. The lock only says what
// was installed from where: enough to offer an update, to notice that an
// installed package was changed by hand, and to ask again before a plugin's
// update does more than its owner agreed to (docs/design/app-center.md 4.2).
package lock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/archive"
)

// Name is the lock file, at the root of a project.
const Name = "kite.lock"

// Version is the format of the lock file this Kite writes.
const Version = 1

// File is a project's lock file.
type File struct {
	LockfileVersion int              `yaml:"lockfileVersion"`
	Themes          map[string]Entry `yaml:"themes,omitempty"`
	Plugins         map[string]Entry `yaml:"plugins,omitempty"`

	// Other keeps the keys this Kite does not know, such as the Kite version
	// a later one pins, so that writing the file never drops them.
	Other map[string]any `yaml:",inline"`
}

// Entry records one package installed from an index.
type Entry struct {
	Version string `yaml:"version"`
	// Source is the index the package was installed from.
	Source string `yaml:"source"`
	// Resolved is the address of its archive in that index, and Checksum
	// the archive's sha256, as sha256:<hex>.
	Resolved string `yaml:"resolved"`
	Checksum string `yaml:"checksum"`
	// Tree is the digest of the files it installed; see [Tree].
	Tree string `yaml:"tree"`
	// Granted is what a plugin's site owner agreed it may do.
	Granted *Grant `yaml:"granted,omitempty"`
}

// Grant is what a plugin does to a site: the pieces of code it puts on
// pages, the other sites those have a reader's browser load from, and the
// hooks it runs while the site is built.
type Grant struct {
	Inject int      `yaml:"inject"`
	Loads  []string `yaml:"loads"`
	Hooks  []string `yaml:"hooks"`
}

// Exceeds lists what g does that was not granted: each new site and hook,
// and more code injected. It is empty when g asks for nothing more.
func (g Grant) Exceeds(granted Grant) []string {
	var more []string
	for _, host := range g.Loads {
		if !slices.Contains(granted.Loads, host) {
			more = append(more, "loads from "+host)
		}
	}
	for _, hook := range g.Hooks {
		if !slices.Contains(granted.Hooks, hook) {
			more = append(more, "runs "+hook)
		}
	}
	if g.Inject > granted.Inject {
		more = append(more, fmt.Sprintf("injects %d pieces of code, not %d", g.Inject, granted.Inject))
	}
	return more
}

// Read reads a project's lock file. A project without one has an empty one.
func Read(root string) (*File, error) {
	data, err := os.ReadFile(filepath.Join(root, Name))
	if errors.Is(err, fs.ErrNotExist) {
		return &File{LockfileVersion: Version}, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse reads a lock file's bytes.
func Parse(data []byte) (*File, error) {
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", Name, err)
	}
	switch {
	case f.LockfileVersion > Version:
		return nil, fmt.Errorf("%s is format %d, and this Kite reads format %d; upgrade Kite",
			Name, f.LockfileVersion, Version)
	case f.LockfileVersion == 0:
		f.LockfileVersion = Version
	}
	return &f, nil
}

func (f *File) section(kind string) *map[string]Entry {
	if kind == "plugin" {
		return &f.Plugins
	}
	return &f.Themes
}

// Get finds the entry of a theme or a plugin, kind, by its name.
func (f *File) Get(kind, name string) (Entry, bool) {
	e, ok := (*f.section(kind))[name]
	return e, ok
}

// Set records a package.
func (f *File) Set(kind, name string, e Entry) {
	s := f.section(kind)
	if *s == nil {
		*s = make(map[string]Entry)
	}
	if e.Granted != nil {
		e.Granted.Loads = nonNil(e.Granted.Loads)
		e.Granted.Hooks = nonNil(e.Granted.Hooks)
	}
	(*s)[name] = e
}

// Delete forgets a package, and reports whether there was one to forget.
func (f *File) Delete(kind, name string) bool {
	s := f.section(kind)
	if _, ok := (*s)[name]; !ok {
		return false
	}
	delete(*s, name)
	return true
}

// Empty reports whether the file records nothing, so that it need not be
// kept.
func (f *File) Empty() bool {
	return len(f.Themes) == 0 && len(f.Plugins) == 0 && len(f.Other) == 0
}

// header says what the file is to whoever opens it.
const header = "# Written by Kite: where each theme and plugin installed from an index\n" +
	"# came from. Their files are in themes/ and plugins/.\n"

// Bytes is the file as it is written: the same records always make the same
// bytes.
func (f *File) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(header)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Tree is the digest of a package's files: each file's sha256 and path, in
// path order, as sha256sum prints them, hashed again, as sha256:<hex>.
//
// Line endings are read as \n, so a checkout that turned them into \r\n, as
// Git does on Windows, still matches the files that were installed; and what
// a computer adds to a folder on its own, such as .DS_Store, is left out, as
// an archive leaves it out.
func Tree(files map[string][]byte) string {
	h := sha256.New()
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if archive.Ignored(name) {
			continue
		}
		sum := sha256.Sum256(bytes.ReplaceAll(files[name], []byte("\r\n"), []byte("\n")))
		_, _ = fmt.Fprintf(h, "%x  %s\n", sum, name)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// TreeOf is the [Tree] of the files in an installed package's directory.
func TreeOf(dir string) (string, error) {
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return "", err
	}
	return Tree(files), nil
}

// Checksum is how the lock writes an archive's sha256, given in hex.
func Checksum(sha256Hex string) string { return "sha256:" + sha256Hex }

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
