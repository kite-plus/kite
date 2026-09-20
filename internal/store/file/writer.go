package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/content"
)

// tmpDir holds partially written files. It sits inside the project so that the
// final rename is always within one filesystem, which is what makes the write
// atomic.
const tmpDir = ".kite/tmp"

const (
	filePerm = 0o644
	dirPerm  = 0o755
)

// Writer implements content.Writer on top of markdown files.
//
// Markdown files are the source of truth: nothing here consults a database,
// and every write lands in the working tree where the user's editor and git
// can see it immediately.
type Writer struct {
	root  string
	types *content.Registry
	codec *Codec
	now   func() time.Time

	mu sync.Mutex
}

// NewWriter returns a writer rooted at a project directory.
func NewWriter(root string, types *content.Registry) *Writer {
	return &Writer{
		root:  root,
		types: types,
		codec: NewCodec(types),
		now:   time.Now,
	}
}

// Apply executes a change set.
//
// Operations are applied in order and the result lists exactly the files that
// were touched, which is what the git publisher stages: never "git add .", only
// these paths (see docs/design/architecture.md 16.3).
func (w *Writer) Apply(ctx context.Context, cs content.ChangeSet) (content.Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var res content.Result
	if cs.IsEmpty() {
		return res, nil
	}

	located, err := w.locate()
	if err != nil {
		return res, err
	}

	for _, op := range cs.Ops {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		switch o := op.(type) {
		case content.PutContent:
			err = w.putContent(o, located, &res)
		case content.DeleteContent:
			err = w.deleteContent(o, located, &res)
		case content.MoveContent:
			err = w.moveContent(o, located, &res)
		case content.PutMedia:
			err = w.putMedia(o, located, &res)
		case content.DeleteMedia:
			err = w.deleteMedia(o, located, &res)
		default:
			err = fmt.Errorf("file store: unsupported operation %q", op.Kind())
		}
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// locate indexes the current tree by ID.
//
// This rescans on every Apply, which is fine at M0 volumes and is replaced by
// the derived index once that lands; Writer only depends on the map, not on how
// it was built.
func (w *Writer) locate() (map[content.ID]*Entry, error) {
	scan, err := NewScanner(w.root, w.types).Scan()
	if err != nil {
		return nil, err
	}
	out := make(map[content.ID]*Entry, len(scan.Entries))
	for _, e := range scan.Entries {
		if e.Item.ID != "" {
			out[e.Item.ID] = e
		}
	}
	return out, nil
}

func (w *Writer) putContent(op content.PutContent, located map[content.ID]*Entry, res *content.Result) error {
	item := op.Content
	if item == nil {
		return errors.New("file store: put_content with no content")
	}
	t := w.types.Get(item.Kind)
	if t == nil {
		return fmt.Errorf("file store: unknown content kind %q", item.Kind)
	}

	if item.ID == "" {
		item.ID = content.NewID()
	}
	now := w.now().UTC().Truncate(time.Second)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if item.Slug == "" {
		item.Slug = Slugify(item.Title)
	}
	if err := item.Validate(); err != nil {
		return err
	}

	// The ID identifies the item, but the locator is its physical address. Both
	// are consulted: a known ID wins, and a caller-supplied locator covers the
	// case of adopting a file that exists but carries no ID yet.
	if e := located[item.ID]; e != nil {
		item.Locator = e.Locator
	}
	if item.Locator == "" {
		key, err := w.freeKey(t, item.Slug)
		if err != nil {
			return err
		}
		item.Locator = LocatorFor(t, key)
	}

	src := SourcePath(t, item.Locator)
	current, err := os.ReadFile(abs(w.root, src))
	switch {
	case err == nil:
		if err := checkRevision(item.ID, op.IfRevision, RevisionOf(current), current); err != nil {
			return err
		}
	case errors.Is(err, fs.ErrNotExist):
		current = nil
		if op.IfRevision != "" {
			return fmt.Errorf("%w: %s does not exist", content.ErrNotFound, item.Locator)
		}
	default:
		return fmt.Errorf("file store: read %s: %w", src, err)
	}

	data, err := w.codec.Encode(t, item, current)
	if err != nil {
		return err
	}
	if err := w.write(src, data); err != nil {
		return err
	}

	item.Revision = RevisionOf(data)
	res.Revision = item.Revision
	res.IDs = append(res.IDs, item.ID)
	appendUnique(&res.Written, src)
	return nil
}

func (w *Writer) deleteContent(op content.DeleteContent, located map[content.ID]*Entry, res *content.Result) error {
	e, ok := located[op.ID]
	if !ok {
		return fmt.Errorf("%w: %s", content.ErrNotFound, op.ID)
	}
	data, err := os.ReadFile(abs(w.root, e.Path))
	if err != nil {
		return fmt.Errorf("file store: read %s: %w", e.Path, err)
	}
	if err := checkRevision(op.ID, op.IfRevision, RevisionOf(data), data); err != nil {
		return err
	}

	if op.Soft {
		now := w.now().UTC().Truncate(time.Second)
		e.Item.DeletedAt = &now
		e.Item.UpdatedAt = now
		encoded, err := w.codec.Encode(e.Type, e.Item, data)
		if err != nil {
			return err
		}
		if err := w.write(e.Path, encoded); err != nil {
			return err
		}
		res.Revision = RevisionOf(encoded)
		appendUnique(&res.Written, e.Path)
		return nil
	}

	// A bundle owns its media, so removing the item removes the directory.
	if e.Type.Layout == content.LayoutBundle {
		removed, err := w.removeTree(string(e.Locator))
		if err != nil {
			return err
		}
		for _, p := range removed {
			appendUnique(&res.Removed, p)
		}
		return nil
	}
	if err := w.remove(e.Path); err != nil {
		return err
	}
	appendUnique(&res.Removed, e.Path)
	return nil
}

func (w *Writer) moveContent(op content.MoveContent, located map[content.ID]*Entry, res *content.Result) error {
	e, ok := located[op.ID]
	if !ok {
		return fmt.Errorf("%w: %s", content.ErrNotFound, op.ID)
	}
	if e.Locator == op.To {
		return nil
	}
	from := abs(w.root, string(e.Locator))
	to := abs(w.root, string(op.To))
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("file store: %s already exists", op.To)
	}
	if err := os.MkdirAll(filepath.Dir(to), dirPerm); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("file store: move %s: %w", e.Locator, err)
	}

	if op.WriteAlias {
		e.Item.Aliases = append(e.Item.Aliases, e.Item.Slug)
	}
	e.Item.Locator = op.To
	newSrc := SourcePath(e.Type, op.To)
	data, err := os.ReadFile(abs(w.root, newSrc))
	if err != nil {
		return err
	}
	encoded, err := w.codec.Encode(e.Type, e.Item, data)
	if err != nil {
		return err
	}
	if err := w.write(newSrc, encoded); err != nil {
		return err
	}
	appendUnique(&res.Removed, SourcePath(e.Type, e.Locator))
	appendUnique(&res.Written, newSrc)
	res.Revision = RevisionOf(encoded)
	return nil
}

func (w *Writer) putMedia(op content.PutMedia, located map[content.ID]*Entry, res *content.Result) error {
	e, ok := located[op.Owner]
	if !ok {
		return fmt.Errorf("%w: media owner %s", content.ErrNotFound, op.Owner)
	}
	dir := MediaDir(e.Type, e.Locator)
	if dir == "" {
		return fmt.Errorf("file store: content type %q stores items as single files and cannot hold media", e.Type.Kind)
	}
	name := filepath.Base(filepath.FromSlash(op.Name))
	if name == "." || name == string(filepath.Separator) {
		return fmt.Errorf("file store: invalid media name %q", op.Name)
	}
	target := path.Join(dir, name)
	if err := w.write(target, op.Data); err != nil {
		return err
	}
	appendUnique(&res.Written, target)
	return nil
}

func (w *Writer) deleteMedia(op content.DeleteMedia, located map[content.ID]*Entry, res *content.Result) error {
	e, ok := located[op.Owner]
	if !ok {
		return fmt.Errorf("%w: media owner %s", content.ErrNotFound, op.Owner)
	}
	dir := MediaDir(e.Type, e.Locator)
	if dir == "" {
		return nil
	}
	target := path.Join(dir, filepath.Base(filepath.FromSlash(op.Name)))
	if err := w.remove(target); err != nil {
		return err
	}
	appendUnique(&res.Removed, target)
	return nil
}

// freeKey picks an unused bundle directory or file name for a new item.
func (w *Writer) freeKey(t *content.Type, slug string) (string, error) {
	base := Slugify(slug)
	if base == "" {
		base = "untitled"
	}
	for i := range 1000 {
		key := base
		if i > 0 {
			key = base + "-" + strconv.Itoa(i+1)
		}
		_, err := os.Stat(abs(w.root, string(LocatorFor(t, key))))
		if errors.Is(err, fs.ErrNotExist) {
			return key, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("file store: no free name for %q", base)
}

// write stores data at a project relative path atomically: the bytes land in a
// temporary file on the same filesystem, are flushed, and only then replace the
// target. A crash therefore leaves either the old file or the new one, never a
// truncated mix.
func (w *Writer) write(relPath string, data []byte) error {
	target := abs(w.root, relPath)
	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return err
	}
	staging := abs(w.root, tmpDir)
	if err := os.MkdirAll(staging, dirPerm); err != nil {
		return err
	}

	f, err := os.CreateTemp(staging, "write-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, filePerm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("file store: write %s: %w", relPath, err)
	}
	return syncDir(filepath.Dir(target))
}

func (w *Writer) remove(relPath string) error {
	if err := os.Remove(abs(w.root, relPath)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("file store: remove %s: %w", relPath, err)
	}
	return syncDir(filepath.Dir(abs(w.root, relPath)))
}

// removeTree deletes a directory and reports every file it held, so the
// publisher can stage the deletions precisely.
func (w *Writer) removeTree(relDir string) ([]string, error) {
	root := abs(w.root, relDir)
	var removed []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		r, err := rel(w.root, p)
		if err != nil {
			return err
		}
		removed = append(removed, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(root); err != nil {
		return nil, fmt.Errorf("file store: remove %s: %w", relDir, err)
	}
	slices.Sort(removed)
	return removed, syncDir(filepath.Dir(root))
}

// syncDir flushes a directory entry so a rename survives a crash.
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return nil // best effort: not all platforms allow opening a directory
	}
	defer func() { _ = f.Close() }()
	_ = f.Sync()
	return nil
}

func checkRevision(id content.ID, want, got content.Revision, current []byte) error {
	if want == "" || want == got {
		return nil
	}
	return &content.ConflictError{
		ID:       id,
		Expected: want,
		Actual:   got,
		Theirs:   current,
	}
}

func appendUnique(dst *[]string, p string) {
	if !slices.Contains(*dst, p) {
		*dst = append(*dst, p)
	}
}
