package site

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"

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
	return nil
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
