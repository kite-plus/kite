package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/frontmatter"
)

// tmpDir holds partially written files. It sits inside the project so that the
// final rename is always within one filesystem, which is what makes the write
// atomic.
const tmpDir = ".kite/tmp"

const (
	filePerm = 0o644
	dirPerm  = 0o755
)

// maxFreeNames bounds the search for an unused name, so a directory that
// somehow cannot accept one fails loudly rather than spinning.
const maxFreeNames = 1000

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
	if err := w.precheck(cs, located); err != nil {
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
		case content.RestoreContent:
			err = w.restoreContent(o, located, &res)
		case content.MoveContent:
			err = w.moveContent(o, located, &res)
		case content.PutMedia:
			err = w.putMedia(o, located, &res)
		case content.DeleteMedia:
			err = w.deleteMedia(o, located, &res)
		case content.PutSettings:
			err = w.putSettings(o, &res)
		case content.ChangeTerm:
			err = w.changeTerm(o, located, &res)
		case content.PutTheme:
			err = w.putTheme(o, &res)
		case content.DeleteTheme:
			err = w.deleteTheme(o, &res)
		case content.PutPlugin:
			err = w.putPlugin(o, &res)
		case content.DeletePlugin:
			err = w.deletePlugin(o, &res)
		default:
			err = fmt.Errorf("file store: unsupported operation %q", op.Kind())
		}
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// precheck compares every revision a set names with the file on disk before
// anything is written, so a set that would conflict partway through, such as
// a term renamed across many items, leaves every file as it was rather than
// half changed. Each operation still checks its own revision as it runs.
func (w *Writer) precheck(cs content.ChangeSet, located map[content.ID]*Entry) error {
	checked := make(map[content.ID]bool)
	for _, op := range cs.Ops {
		var id content.ID
		var want content.Revision
		switch o := op.(type) {
		case content.PutContent:
			if o.Content != nil {
				id, want = o.Content.ID, o.IfRevision
			}
		case content.DeleteContent:
			id, want = o.ID, o.IfRevision
		case content.RestoreContent:
			id, want = o.ID, o.IfRevision
		case content.ChangeTerm:
			id, want = o.ID, o.IfRevision
		}
		// Only the first operation on an item can be checked here: a later
		// one meets the file the earlier one writes.
		if id == "" || want == "" || checked[id] {
			continue
		}
		checked[id] = true

		e, ok := located[id]
		if !ok || e.Hash == want {
			continue // a missing item is the operation's own error to report
		}
		current, err := os.ReadFile(abs(w.root, e.Path))
		if err != nil {
			return fmt.Errorf("file store: read %s: %w", e.Path, err)
		}
		return checkRevision(id, want, RevisionOf(current), current)
	}
	return nil
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
	if err := w.settle(t, item, located); err != nil {
		return err
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

	w.stamp(item, current)

	if err := item.Validate(); err != nil {
		return err
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

// stamp fills in the timestamps Kite is entitled to write.
//
// Only a file Kite creates gets a created_at, and updated_at is maintained
// only for a document that already declares one. Stamping both on every save
// would add two lines to the diff of a one word fix, which is the complaint
// every git-backed CMS earns first; and a file that never recorded when it
// was updated is telling the truth rather than waiting to be corrected.
func (w *Writer) stamp(item *content.Content, current []byte) {
	now := w.now().UTC().Truncate(time.Second)

	if current == nil {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		return
	}
	if w.codec.Declares(current, keyUpdatedAt) {
		item.UpdatedAt = now
	}
}

// changeTerm rewrites the one taxonomy key of an item's file that holds the
// term. updated_at moves along where the file keeps one, as any edit moves it.
func (w *Writer) changeTerm(op content.ChangeTerm, located map[content.ID]*Entry, res *content.Result) error {
	e, ok := located[op.ID]
	if !ok {
		return fmt.Errorf("%w: %s", content.ErrNotFound, op.ID)
	}
	if !slices.Contains(e.Type.Taxonomies, op.Taxonomy) {
		return fmt.Errorf("%w: %s items have no taxonomy %q", content.ErrInvalid, e.Type.Kind, op.Taxonomy)
	}
	current, err := os.ReadFile(abs(w.root, e.Path))
	if err != nil {
		return fmt.Errorf("file store: read %s: %w", e.Path, err)
	}
	if err := checkRevision(op.ID, op.IfRevision, RevisionOf(current), current); err != nil {
		return err
	}

	data, err := w.codec.ChangeTerm(current, op.Taxonomy, op.Term, op.To, w.now())
	if err != nil {
		return fmt.Errorf("%s: %w", e.Path, err)
	}
	if err := w.write(e.Path, data); err != nil {
		return err
	}
	res.Revision = RevisionOf(data)
	res.IDs = append(res.IDs, op.ID)
	appendUnique(&res.Written, e.Path)
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
		if e.Item.DeletedAt != nil {
			return fmt.Errorf("%w: %s is already deleted", content.ErrInvalid, op.ID)
		}
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

func (w *Writer) restoreContent(op content.RestoreContent, located map[content.ID]*Entry, res *content.Result) error {
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
	if e.Item.DeletedAt == nil {
		return fmt.Errorf("%w: %s is not deleted", content.ErrInvalid, op.ID)
	}
	e.Item.DeletedAt = nil
	e.Item.UpdatedAt = w.now().UTC().Truncate(time.Second)
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

// UploadsDir holds the files that belong to the site rather than to one
// item, such as a logo a theme setting names. It sits in static/, which a
// build publishes at the root of the site.
const UploadsDir = "static/uploads"

func (w *Writer) putMedia(op content.PutMedia, located map[content.ID]*Entry, res *content.Result) error {
	dir := UploadsDir
	if op.Owner != "" {
		e, ok := located[op.Owner]
		if !ok {
			return fmt.Errorf("%w: media owner %s", content.ErrNotFound, op.Owner)
		}
		dir = MediaDir(e.Type, e.Locator)
		if dir == "" {
			return fmt.Errorf("file store: content type %q stores items as single files and cannot hold media", e.Type.Kind)
		}
	}
	name := filepath.Base(filepath.FromSlash(op.Name))
	if name == "." || name == string(filepath.Separator) {
		return fmt.Errorf("file store: invalid media name %q", op.Name)
	}
	target := path.Join(dir, name)
	if !op.Replace {
		free, err := w.freeMediaName(dir, name)
		if err != nil {
			return err
		}
		target = free
	}
	if err := w.write(target, op.Data); err != nil {
		return err
	}
	appendUnique(&res.Written, target)
	return nil
}

// putSettings changes values in the project's configuration file.
func (w *Writer) putSettings(op content.PutSettings, res *content.Result) error {
	if len(op.Values) == 0 {
		return nil
	}
	current, err := os.ReadFile(abs(w.root, ConfigName))
	if err != nil {
		return fmt.Errorf("file store: read %s: %w", ConfigName, err)
	}
	actual := RevisionOf(current)
	if op.IfRevision != "" && op.IfRevision != actual {
		return &content.ConflictError{
			ID: content.ID(ConfigName), Expected: op.IfRevision, Actual: actual,
		}
	}

	doc, err := frontmatter.ParseYAML(current)
	if err != nil {
		return fmt.Errorf("file store: %s: %w", ConfigName, err)
	}
	// Sorted so that one change set always produces the same bytes, whatever
	// order the caller happened to build its map in.
	for _, dotted := range slices.Sorted(maps.Keys(op.Values)) {
		parts := strings.Split(dotted, ".")
		if len(parts) < 1 || slices.Contains(parts, "") {
			return fmt.Errorf("%w: setting path %q", content.ErrInvalid, dotted)
		}
		section, key := parts[:len(parts)-1], parts[len(parts)-1]
		if op.Values[dotted] == nil {
			if err := doc.DeleteNested(section, key); err != nil {
				return err
			}
			continue
		}
		if err := doc.SetNested(section, key, op.Values[dotted]); err != nil {
			return err
		}
	}
	if !doc.Dirty() {
		return nil
	}

	data, err := doc.Bytes()
	if err != nil {
		return err
	}
	if err := w.write(ConfigName, data); err != nil {
		return err
	}
	res.Revision = RevisionOf(data)
	appendUnique(&res.Written, ConfigName)
	return nil
}

// ThemesDir is where installed themes live, one directory each.
const ThemesDir = "themes"

// PluginsDir holds installed plugins, one directory each.
const PluginsDir = "plugins"

// putTheme writes a theme's files into its directory, and on a replace takes
// away the files the old version had that the new one does not.
func (w *Writer) putTheme(op content.PutTheme, res *content.Result) error {
	if !content.ValidThemeName(op.Name) {
		return fmt.Errorf("%w: theme name %q", content.ErrInvalid, op.Name)
	}
	return w.putPackage("theme", path.Join(ThemesDir, op.Name), op.Files, op.Replace, res)
}

// putPlugin writes a plugin's files into its directory, as putTheme does.
func (w *Writer) putPlugin(op content.PutPlugin, res *content.Result) error {
	if !config.ValidPluginID(op.ID) {
		return fmt.Errorf("%w: plugin id %q", content.ErrInvalid, op.ID)
	}
	return w.putPackage("plugin", path.Join(PluginsDir, op.ID), op.Files, op.Replace, res)
}

// putPackage writes the files of a theme or a plugin, kind, into dir.
func (w *Writer) putPackage(kind, dir string, files map[string][]byte, replace bool, res *content.Result) error {
	if len(files) == 0 {
		return fmt.Errorf("%w: %s %s has no files", content.ErrInvalid, kind, path.Base(dir))
	}
	names := slices.Sorted(maps.Keys(files))
	for _, name := range names {
		if !fs.ValidPath(name) || name == "." {
			return fmt.Errorf("%w: %s file %q", content.ErrInvalid, kind, name)
		}
	}

	old, err := w.packageFiles(kind, dir)
	if err != nil {
		return err
	}
	if len(old) > 0 && !replace {
		return fmt.Errorf("%w: %s %s is already installed", content.ErrInvalid, kind, path.Base(dir))
	}

	for _, name := range names {
		target := path.Join(dir, name)
		if err := w.write(target, files[name]); err != nil {
			return err
		}
		appendUnique(&res.Written, target)
	}
	for _, target := range old {
		if _, kept := files[strings.TrimPrefix(target, dir+"/")]; kept {
			continue
		}
		if err := w.remove(target); err != nil {
			return err
		}
		appendUnique(&res.Removed, target)
	}
	pruneEmpty(abs(w.root, dir))
	return nil
}

// packageFiles lists the files of an installed theme or plugin, kind, none
// when it is not installed. A link to one kept elsewhere is refused rather
// than written through, since its files belong to wherever it points.
func (w *Writer) packageFiles(kind, dir string) ([]string, error) {
	info, err := os.Lstat(abs(w.root, dir))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	case info.Mode()&fs.ModeSymlink != 0:
		return nil, fmt.Errorf("%w: %s links to a %s kept elsewhere; change it there", content.ErrInvalid, dir, kind)
	case !info.IsDir():
		return nil, fmt.Errorf("%w: %s is a file, not a %s", content.ErrInvalid, dir, kind)
	}
	var files []string
	err = filepath.WalkDir(abs(w.root, dir), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		r, err := rel(w.root, p)
		if err != nil {
			return err
		}
		files = append(files, r)
		return nil
	})
	return files, err
}

// pruneEmpty removes the directories under root that removals left empty.
func pruneEmpty(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != root {
			dirs = append(dirs, p)
		}
		return nil
	})
	// Deepest first, so a parent is empty by the time it is tried.
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
}

// deleteTheme removes an installed theme. A link to a theme kept elsewhere
// is removed itself, leaving what it pointed at alone.
func (w *Writer) deleteTheme(op content.DeleteTheme, res *content.Result) error {
	if !content.ValidThemeName(op.Name) {
		return fmt.Errorf("%w: theme name %q", content.ErrInvalid, op.Name)
	}
	return w.deletePackage("theme", path.Join(ThemesDir, op.Name), res)
}

// deletePlugin removes a plugin's directory, as deleteTheme does.
func (w *Writer) deletePlugin(op content.DeletePlugin, res *content.Result) error {
	if !config.ValidPluginID(op.ID) {
		return fmt.Errorf("%w: plugin id %q", content.ErrInvalid, op.ID)
	}
	return w.deletePackage("plugin", path.Join(PluginsDir, op.ID), res)
}

// deletePackage removes the directory of a theme or plugin, kind.
func (w *Writer) deletePackage(kind, dir string, res *content.Result) error {
	info, err := os.Lstat(abs(w.root, dir))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %s %s", content.ErrNotFound, kind, path.Base(dir))
	case err != nil:
		return err
	case info.Mode()&fs.ModeSymlink != 0 || !info.IsDir():
		if err := w.remove(dir); err != nil {
			return err
		}
		appendUnique(&res.Removed, dir)
		return nil
	}
	removed, err := w.removeTree(dir)
	if err != nil {
		return err
	}
	for _, p := range removed {
		appendUnique(&res.Removed, p)
	}
	return nil
}

// freeMediaName returns a path in dir that nothing occupies, suffixing the
// base name until one is free.
func (w *Writer) freeMediaName(dir, name string) (string, error) {
	ext := path.Ext(name)
	base, err := firstFree(strings.TrimSuffix(name, ext), func(candidate string) (bool, error) {
		_, err := os.Stat(abs(w.root, path.Join(dir, candidate+ext)))
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return "", err
	}
	return path.Join(dir, base+ext), nil
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

// untitledKey names an item whose title yields no slug at all, such as an
// empty one or a title made only of punctuation.
const untitledKey = "untitled"

// slugKey is the uniqueness rule the read model enforces, expressed here so
// the store can uphold it before it writes rather than after.
type slugKey struct {
	kind   content.Kind
	slug   string
	locale string
}

// takenSlugs maps every slug in the tree to the item holding it, skipping the
// item being written so that saving an item does not collide with itself.
func takenSlugs(located map[content.ID]*Entry, self content.ID) map[slugKey]*Entry {
	out := make(map[slugKey]*Entry, len(located))
	for id, e := range located {
		if id == self {
			continue
		}
		out[slugKey{e.Item.Kind, e.Item.Slug, e.Item.Locale}] = e
	}
	return out
}

// settle decides where an item's bytes live and what slug they carry.
//
// The read model requires (kind, slug, locale) to be unique, so a collision is
// resolved here, before anything is written. Finding it afterwards would leave
// a file on disk that the index refuses to load, and the request that wrote it
// would report the item it had just created as missing.
//
// A slug the author typed is their URL, so a collision there is refused rather
// than quietly altered. A slug Kite derives from the title is Kite's own: it
// is suffixed until it is free, and a bundle being placed for the first time
// takes the same suffix, so that folder, slug and URL agree.
func (w *Writer) settle(t *content.Type, item *content.Content, located map[content.ID]*Entry) error {
	// The ID identifies the item, but the locator is its physical address.
	// Both are consulted: a known ID wins, and a caller-supplied locator covers
	// the case of adopting a file that exists but carries no ID yet.
	stored := located[item.ID]
	if stored != nil {
		item.Locator = stored.Locator
	}

	taken := takenSlugs(located, item.ID)
	holder := func(slug string) content.Locator {
		e, ok := taken[slugKey{item.Kind, slug, item.Locale}]
		if !ok {
			return ""
		}
		return e.Locator
	}

	if item.Slug != "" {
		// A collision this write did not create must not block an unrelated
		// edit: the file already sits there under that slug, and refusing the
		// save would leave the author no way to change anything else about it.
		kept := stored != nil && stored.Item.Slug == item.Slug && stored.Item.Locale == item.Locale
		if at := holder(item.Slug); at != "" && !kept {
			return fmt.Errorf("%w: slug %q is already used by %s", content.ErrInvalid, item.Slug, at)
		}
		if item.Locator == "" {
			key, err := w.freeKey(t, item.Slug, nil)
			if err != nil {
				return err
			}
			item.Locator = LocatorFor(t, key)
		}
		return nil
	}

	base := Slugify(item.Title)
	if base == "" {
		base = untitledKey
	}
	free := func(slug string) bool { return holder(slug) == "" }

	// An item that already has an address keeps it: slug and path are
	// independent, and a retitled post must not move (see D2).
	if item.Locator != "" {
		slug, err := firstFree(base, func(s string) (bool, error) { return free(s), nil })
		if err != nil {
			return err
		}
		item.Slug = slug
		return nil
	}

	key, err := w.freeKey(t, base, free)
	if err != nil {
		return err
	}
	item.Slug, item.Locator = key, LocatorFor(t, key)
	return nil
}

// freeKey picks an unused bundle directory or file name for a new item. A key
// is only free when the slug it would yield is free too, when slugFree says so.
func (w *Writer) freeKey(t *content.Type, slug string, slugFree func(string) bool) (string, error) {
	base := Slugify(slug)
	if base == "" {
		base = untitledKey
	}
	return firstFree(base, func(key string) (bool, error) {
		if slugFree != nil && !slugFree(key) {
			return false, nil
		}
		_, err := os.Stat(abs(w.root, string(LocatorFor(t, key))))
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	})
}

// firstFree walks base, base-2, base-3 and so on, returning the first
// candidate ok accepts.
func firstFree(base string, ok func(string) (bool, error)) (string, error) {
	for i := range maxFreeNames {
		candidate := base
		if i > 0 {
			candidate = base + "-" + strconv.Itoa(i+1)
		}
		free, err := ok(candidate)
		if err != nil {
			return "", err
		}
		if free {
			return candidate, nil
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
