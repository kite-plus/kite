package build

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Emitter collects a build's output and publishes it in one step.
//
// Files are written to a staging directory and only swapped into place when
// the whole build has succeeded. A failed build therefore leaves the previous
// site intact rather than a half-replaced mixture of old and new pages.
type Emitter struct {
	outDir   string
	stageDir string

	mu      sync.Mutex
	written []string
}

// NewEmitter prepares an emitter writing into outDir.
func NewEmitter(outDir string) (*Emitter, error) {
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return nil, err
	}
	stage := abs + ".tmp"
	if err := os.RemoveAll(stage); err != nil {
		return nil, fmt.Errorf("build: clear staging directory: %w", err)
	}
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return nil, fmt.Errorf("build: create staging directory: %w", err)
	}
	return &Emitter{outDir: abs, stageDir: stage}, nil
}

// Write stores one output file. The path is site-relative and slash separated.
//
// A path that would leave the output directory is refused rather than clamped:
// silently rewriting it would put the file somewhere the caller did not mean,
// and no legitimate build target ever asks for one.
func (e *Emitter) Write(rel string, data []byte) error {
	clean := path.Clean(rel)
	if clean == "" || clean == "." || path.IsAbs(clean) ||
		clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("build: refusing to write outside the output directory: %q", rel)
	}

	target := filepath.Join(e.stageDir, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("build: write %s: %w", clean, err)
	}

	e.mu.Lock()
	e.written = append(e.written, clean)
	e.mu.Unlock()
	return nil
}

// CopyTree copies a directory of unprocessed files, such as static assets.
func (e *Emitter) CopyTree(src fs.FS, prefix string) error {
	if src == nil {
		return nil
	}
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		return e.Write(path.Join(prefix, p), data)
	})
}

// Files returns every path written, sorted.
func (e *Emitter) Files() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := slices.Clone(e.written)
	slices.Sort(out)
	return out
}

// Commit swaps the staged output into place.
func (e *Emitter) Commit() error {
	previous := e.outDir + ".prev"
	if err := os.RemoveAll(previous); err != nil {
		return err
	}

	// Move the old output aside rather than deleting it first, so the window
	// in which no output exists is as short as a rename.
	switch _, err := os.Stat(e.outDir); {
	case err == nil:
		if err := os.Rename(e.outDir, previous); err != nil {
			return fmt.Errorf("build: move previous output aside: %w", err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if err := os.Rename(e.stageDir, e.outDir); err != nil {
		// Put the previous output back rather than leaving nothing behind.
		_ = os.Rename(previous, e.outDir)
		return fmt.Errorf("build: publish output: %w", err)
	}
	return os.RemoveAll(previous)
}

// Discard removes the staged output, for a build that failed.
func (e *Emitter) Discard() error { return os.RemoveAll(e.stageDir) }
