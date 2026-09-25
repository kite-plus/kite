package serve

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// settle is how long a burst of filesystem events is allowed to continue
// before the server reacts.
//
// Saving one file in an editor produces several events, and a checkout
// produces thousands; reacting to each one would rebuild the routing table
// dozens of times for a single keystroke.
const settle = 120 * time.Millisecond

// watcher reports that something under the project changed.
//
// It is only a latency optimisation. What it triggers is a full reconcile of
// the index, which compares the tree on disk rather than trusting the events:
// editors save by writing a temporary file and renaming it, which destroys the
// inode being watched, so a watcher alone silently misses changes.
type watcher struct {
	fsw  *fsnotify.Watcher
	root string
	log  *slog.Logger
}

func newWatcher(root string, log *slog.Logger) (*watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &watcher{fsw: fsw, root: root, log: log}

	// Directories are watched, not files: a rename replaces the file's inode
	// but leaves the directory's intact.
	for _, dir := range []string{"content", "layouts", "themes", "static"} {
		w.addTree(filepath.Join(root, dir))
	}
	if entries, err := os.ReadDir(filepath.Join(root, "themes")); err == nil {
		for _, e := range entries {
			if e.Type()&fs.ModeSymlink != 0 {
				w.addLinkedTheme(filepath.Join(root, "themes", e.Name()))
			}
		}
	}
	// The config and the git HEAD are single files whose directory is the
	// project root, which is watched non-recursively for exactly this.
	_ = fsw.Add(root)
	_ = fsw.Add(filepath.Join(root, ".git"))

	return w, nil
}

// addLinkedTheme watches a theme that themes/ links to rather than holds,
// as one being written in its own repository is: a walk does not follow the
// link. Only the theme's own folders are walked, since its repository often
// holds the very site being served.
func (w *watcher) addLinkedTheme(link string) {
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		return
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		return
	}
	_ = w.fsw.Add(target)
	for _, dir := range []string{"layouts", "i18n", "static", "assets"} {
		w.addTree(filepath.Join(target, dir))
	}
}

func (w *watcher) addTree(root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if name := d.Name(); strings.HasPrefix(name, ".") && p != root {
			return fs.SkipDir
		}
		_ = w.fsw.Add(p)
		return nil
	})
}

// Close releases the watch.
func (w *watcher) Close() error { return w.fsw.Close() }

// run calls onChange once per settled burst of events.
func (w *watcher) run(ctx context.Context, onChange func()) {
	var timer *time.Timer
	var fire <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if !w.interesting(event) {
				continue
			}
			// A new directory has to be watched too, or content created
			// inside it would never be noticed.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Lstat(event.Name); err == nil && info.Mode()&fs.ModeSymlink != 0 {
					if filepath.Dir(event.Name) == filepath.Join(w.root, "themes") {
						w.addLinkedTheme(event.Name)
					}
				} else if err == nil && info.IsDir() {
					w.addTree(event.Name)
				}
			}
			if timer == nil {
				timer = time.NewTimer(settle)
				fire = timer.C
			} else {
				timer.Reset(settle)
			}

		case <-fire:
			timer, fire = nil, nil
			onChange()

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			w.log.Warn("watch error", "err", err)
		}
	}
}

// interesting filters out the noise a working directory generates.
func (w *watcher) interesting(e fsnotify.Event) bool {
	if e.Op&fsnotify.Chmod != 0 && e.Op == fsnotify.Chmod {
		return false
	}

	base := filepath.Base(e.Name)
	switch {
	case strings.HasSuffix(base, "~"), strings.HasSuffix(base, ".swp"), strings.HasSuffix(base, ".tmp"):
		return false
	case strings.HasPrefix(base, ".#"), strings.HasPrefix(base, "#"):
		return false
	}

	rel, err := filepath.Rel(w.root, e.Name)
	if err != nil {
		return false
	}
	// Everything under .kite is output of the server's own work, and public
	// is where a build writes; reacting to either would loop.
	switch {
	case rel == ".":
		return false
	case strings.HasPrefix(rel, ".kite"), strings.HasPrefix(rel, "public"):
		return false
	case strings.HasPrefix(rel, ".git"):
		// A checkout moves HEAD, which is worth a reconcile; the objects
		// churning underneath it are not.
		return base == "HEAD" || base == "index"
	}
	return true
}
