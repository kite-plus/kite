package file

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kite-plus/kite/internal/content"
)

// RevisionOf derives an item's revision from its source bytes. A content hash
// is used rather than mtime so that the value is stable across checkouts and
// machines.
func RevisionOf(data []byte) content.Revision {
	sum := sha256.Sum256(data)
	return content.Revision("sha256:" + hex.EncodeToString(sum[:]))
}

// Entry is one content source file found on disk.
type Entry struct {
	Type    *content.Type
	Locator content.Locator
	Path    string // source file, project relative, slash separated
	Size    int64
	ModTime int64 // unix nanoseconds
	Item    *content.Content
}

// Problem records a file that could not be read, so one broken file reports
// clearly instead of failing the whole scan.
type Problem struct {
	Path string
	Err  error
}

func (p Problem) Error() string { return p.Path + ": " + p.Err.Error() }

// Scanner walks the content tree.
type Scanner struct {
	root  string
	types *content.Registry
	codec *Codec
}

// NewScanner returns a scanner rooted at a project directory.
func NewScanner(root string, types *content.Registry) *Scanner {
	return &Scanner{root: root, types: types, codec: NewCodec(types)}
}

// Result is the outcome of a scan.
type Result struct {
	Entries  []*Entry
	Problems []Problem
}

// Err joins every problem into one error, or returns nil.
func (r *Result) Err() error {
	if len(r.Problems) == 0 {
		return nil
	}
	errs := make([]error, 0, len(r.Problems))
	for _, p := range r.Problems {
		errs = append(errs, p)
	}
	return errors.Join(errs...)
}

// Scan reads every content source file under the project's content directory.
//
// Entries come back sorted by path so that any output derived from a scan is
// deterministic, which is a precondition for reproducible builds.
func (s *Scanner) Scan() (*Result, error) {
	res := &Result{}
	for _, t := range s.types.Types() {
		dir := abs(s.root, path.Join(ContentDir, t.Dir))
		info, err := os.Stat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", t.Dir, err)
		}
		if !info.IsDir() {
			continue
		}
		if err := s.scanType(t, dir, res); err != nil {
			return nil, err
		}
	}

	slices.SortFunc(res.Entries, func(a, b *Entry) int { return strings.Compare(a.Path, b.Path) })
	if err := s.checkDuplicateIDs(res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Scanner) scanType(t *content.Type, dir string, res *Result) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && p != dir {
				return fs.SkipDir
			}
			return nil
		}
		if !IsMarkdown(d.Name()) || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		relPath, err := rel(s.root, p)
		if err != nil {
			return err
		}

		// In a bundle layout only index.md carries the item; other markdown
		// files inside the bundle are just resources.
		loc := content.Locator(relPath)
		if t.Layout == content.LayoutBundle {
			if d.Name() != BundleIndex {
				return nil
			}
			loc = content.Locator(path.Dir(relPath))
		}

		data, err := os.ReadFile(p)
		if err != nil {
			res.Problems = append(res.Problems, Problem{Path: relPath, Err: err})
			return nil
		}
		item, err := s.codec.Decode(t, loc, data)
		if err != nil {
			res.Problems = append(res.Problems, Problem{Path: relPath, Err: err})
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		res.Entries = append(res.Entries, &Entry{
			Type:    t,
			Locator: loc,
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			Item:    item,
		})
		return nil
	})
}

// checkDuplicateIDs reports items sharing an ID.
//
// Copying a bundle directory is a normal thing for a user to do and it
// duplicates the ID in the copy, so this is always surfaced as an error rather
// than silently repaired: only the user knows which copy should keep the ID.
func (s *Scanner) checkDuplicateIDs(res *Result) error {
	seen := make(map[content.ID][]string)
	for _, e := range res.Entries {
		if e.Item.ID == "" {
			continue
		}
		seen[e.Item.ID] = append(seen[e.Item.ID], e.Path)
	}
	var errs []error
	for id, paths := range seen {
		if len(paths) < 2 {
			continue
		}
		slices.Sort(paths)
		errs = append(errs, fmt.Errorf("%w: %s is claimed by %s", content.ErrDuplicateID, id, strings.Join(paths, ", ")))
	}
	if len(errs) == 0 {
		return nil
	}
	slices.SortFunc(errs, func(a, b error) int { return strings.Compare(a.Error(), b.Error()) })
	return errors.Join(errs...)
}

// MissingIDs returns the entries that carry no ID yet, which is what a
// hand-authored file looks like before kite doctor has visited it.
func (r *Result) MissingIDs() []*Entry {
	var out []*Entry
	for _, e := range r.Entries {
		if e.Item.ID == "" {
			out = append(out, e)
		}
	}
	return out
}
