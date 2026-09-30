package site

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/store/file"
)

// ownDirs are the directories of a project that a build reads or that Kite
// keeps its state in, the account included.
var ownDirs = []string{file.ContentDir, "static", theme.LayoutsDir, ThemesDir, plugin.Dir, ".kite", ".git"}

// checkOutDir refuses an output directory that a build would do damage with.
//
// A build swaps its output in whole and deletes what was there, so an output
// that is, or holds, the project or anything the build reads would take it
// along. An output inside one of the project's own directories would be read
// back by the next build, or mixed in with what Kite keeps.
func (s *Site) checkOutDir(dir string) error {
	out, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(out); err == nil && !info.IsDir() {
		return fmt.Errorf("build: output %s is a file, not a directory", out)
	}
	root := s.Project.Root
	if within(root, out) {
		return fmt.Errorf("build: output %s would replace the project", out)
	}

	// The theme and plugins may be linked in from where they are written,
	// such as their own repositories, outside the project.
	keep := slices.Clone(ownDirs)
	if name := s.Config.Theme.Name; name != "" && name != BuiltinTheme {
		keep = append(keep, path.Join(ThemesDir, name))
	}
	for _, p := range s.Plugins {
		keep = append(keep, path.Join(plugin.Dir, p.Manifest.ID))
	}
	for _, rel := range keep {
		if within(filepath.Join(root, filepath.FromSlash(rel)), out) {
			return fmt.Errorf("build: output %s would replace the project's %s", out, rel)
		}
	}
	for _, rel := range ownDirs {
		if within(out, filepath.Join(root, rel)) {
			return fmt.Errorf("build: output %s is inside the project's %s; the site needs a directory of its own", out, rel)
		}
	}
	return s.checkOwned(out)
}

// outputsFile lists the directories the project's builds wrote, one absolute
// path a line.
const outputsFile = ".kite/outputs"

// junk are the files a desktop leaves in a folder it shows, which make no
// directory anyone's.
var junk = map[string]bool{".DS_Store": true, "Thumbs.db": true, "desktop.ini": true}

// checkOwned refuses an output that holds files Kite did not write. A build
// replaces its output whole, so a directory named by mistake, such as a home
// folder, would lose everything in it. Kite's own is one a build of the
// project recorded, or, for one built before builds were recorded, one
// holding the sitemap or the feed a build writes for this site.
func (s *Site) checkOwned(out string) error {
	entries, err := os.ReadDir(out)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return !junk[e.Name()] }) {
		return nil
	}
	if s.wrote(out) || s.namesSite(out, "sitemap.xml", "<urlset", "<loc>") || s.namesSite(out, "rss.xml", "<rss", "<link>") {
		return nil
	}
	return fmt.Errorf("build: output %s holds files Kite did not write, and a build replaces its output whole; empty it or choose another directory", out)
}

// wrote reports whether a build of the project recorded dir as its output.
func (s *Site) wrote(dir string) bool {
	data, err := os.ReadFile(filepath.Join(s.Project.Root, filepath.FromSlash(outputsFile)))
	if err != nil {
		return false
	}
	for line := range strings.Lines(string(data)) {
		if sameDir(strings.TrimSpace(line), dir) {
			return true
		}
	}
	return false
}

// namesSite reports whether dir holds an XML file of the kind root names that
// gives an address of this site in the element tag opens.
func (s *Site) namesSite(dir, name, root, tag string) bool {
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	// The first address is near the top.
	head := make([]byte, 64<<10)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	return bytes.Contains(head, []byte(root)) && bytes.Contains(head, []byte(tag+s.Config.Site.BaseURL+"/"))
}

// recordOutput adds dir to the project's record of the outputs its builds
// wrote, leaving out those that are gone, such as an export's.
func (s *Site) recordOutput(dir string) error {
	out, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	record := filepath.Join(s.Project.Root, filepath.FromSlash(outputsFile))
	kept := []string{out}
	if data, err := os.ReadFile(record); err == nil {
		for line := range strings.Lines(string(data)) {
			p := strings.TrimSpace(line)
			if info, err := os.Stat(p); p != "" && !sameDir(p, out) && err == nil && info.IsDir() {
				kept = append(kept, p)
			}
		}
	}
	slices.Sort(kept)
	return writeWhole(record, []byte(strings.Join(kept, "\n")+"\n"))
}

// sameDir reports whether two absolute paths name one directory, under any
// name.
func sameDir(a, b string) bool {
	if a == "" {
		return false
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	x, errX := os.Stat(a)
	y, errY := os.Stat(b)
	return errX == nil && errY == nil && os.SameFile(x, y)
}

// writeWhole writes a file whole or not at all.
func writeWhole(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+"-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), name)
}

// within reports whether the absolute path p is dir or lies inside it. The
// names are compared, for paths that do not exist yet, and then the
// directories themselves: through a symlink, or spelled in another case on a
// disk that ignores case, a path still leads to the same directory.
func within(p, dir string) bool {
	if rel, err := filepath.Rel(dir, p); err == nil && filepath.IsLocal(rel) {
		return true
	}
	target, err := os.Stat(dir)
	if err != nil {
		return false
	}
	at := resolved(p)
	for {
		if info, err := os.Stat(at); err == nil && os.SameFile(info, target) {
			return true
		}
		up := filepath.Dir(at)
		if up == at {
			return false
		}
		at = up
	}
}

// resolved is the deepest part of the absolute path p that exists, with its
// symlinks followed, so that the directories above it are the ones p really
// lies in.
func resolved(p string) string {
	for {
		if followed, err := filepath.EvalSymlinks(p); err == nil {
			return followed
		}
		up := filepath.Dir(p)
		if up == p {
			return p
		}
		p = up
	}
}
