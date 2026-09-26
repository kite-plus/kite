package content

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

var (
	// ErrNotFound is returned when the target of an operation does not exist.
	ErrNotFound = errors.New("content: not found")

	// ErrConflict is returned when IfRevision does not match what is stored,
	// meaning the item changed underneath the caller. The API surfaces it as
	// 409 together with the version that is now stored.
	ErrConflict = errors.New("content: revision conflict")

	// ErrDuplicateID is returned when two items claim the same ID. Copying a
	// bundle directory is a normal user action, so this is always reported and
	// never silently repaired.
	ErrDuplicateID = errors.New("content: duplicate id")

	// ErrInvalid is returned when an item is not well formed. It exists so a
	// caller can tell "what you sent is wrong" from "something broke", which
	// is the difference between a 400 and a 500 and cannot be recovered by
	// reading the message.
	ErrInvalid = errors.New("content: invalid")
)

// ConflictError reports an edit made against a version that has since been
// replaced.
//
// Theirs is what is stored now. There is no base: the file store keeps no
// history to read one from, and the caller that made the edit still holds the
// version it started from. Carrying a field the store can never fill would
// only invite callers to rely on it.
type ConflictError struct {
	ID       ID
	Expected Revision
	Actual   Revision
	Theirs   []byte
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("content %s: revision conflict (expected %s, found %s)", e.ID, e.Expected, e.Actual)
}

func (e *ConflictError) Unwrap() error { return ErrConflict }

// OpKind names the kind of an [Op].
type OpKind string

const (
	OpPutContent     OpKind = "put_content"
	OpDeleteContent  OpKind = "delete_content"
	OpRestoreContent OpKind = "restore_content"
	OpMoveContent    OpKind = "move_content"
	OpPutMedia       OpKind = "put_media"
	OpDeleteMedia    OpKind = "delete_media"
	OpPutSettings    OpKind = "put_settings"
	OpChangeTerm     OpKind = "change_term"
	OpPutTheme       OpKind = "put_theme"
	OpDeleteTheme    OpKind = "delete_theme"
	OpPutPlugin      OpKind = "put_plugin"
	OpDeletePlugin   OpKind = "delete_plugin"
)

// Op is a single typed operation inside a [ChangeSet].
type Op interface {
	Kind() OpKind
	// Describe returns a short human readable target, used in commit messages
	// and logs.
	Describe() string
}

// PutContent creates or replaces an item.
type PutContent struct {
	Content *Content
	// IfRevision must match the stored revision. It is empty when creating.
	IfRevision Revision
}

func (o PutContent) Kind() OpKind     { return OpPutContent }
func (o PutContent) Describe() string { return string(o.Content.Kind) + "/" + o.Content.Slug }

// DeleteContent removes an item and, for bundle layouts, everything it owns.
type DeleteContent struct {
	ID         ID
	IfRevision Revision
	// Soft records a deletion timestamp instead of removing the bytes.
	Soft bool
}

func (o DeleteContent) Kind() OpKind     { return OpDeleteContent }
func (o DeleteContent) Describe() string { return string(o.ID) }

type RestoreContent struct {
	ID         ID
	IfRevision Revision
}

func (o RestoreContent) Kind() OpKind     { return OpRestoreContent }
func (o RestoreContent) Describe() string { return string(o.ID) }

// MoveContent relocates an item's bytes. Changing a slug does not imply a move:
// slug, path and URL are independent (see D2). A move is only ever performed
// when the user explicitly asks for one.
type MoveContent struct {
	ID ID
	To Locator
	// WriteAlias records the previous URL so it keeps redirecting.
	WriteAlias bool
}

func (o MoveContent) Kind() OpKind     { return OpMoveContent }
func (o MoveContent) Describe() string { return string(o.ID) + " -> " + string(o.To) }

// PutMedia writes a media file, usually into the owning item's bundle. With
// no owner it belongs to the site, as a logo a theme setting names does.
type PutMedia struct {
	Owner ID
	Name  string
	Data  []byte

	// Replace overwrites a file of the same name. Without it the store picks
	// a free name instead, because two screenshots an author drags in are
	// both called screenshot.png and losing one of them is not a reasonable
	// reading of "put this here". The name actually used is reported in
	// Result.Written.
	Replace bool
}

func (o PutMedia) Kind() OpKind     { return OpPutMedia }
func (o PutMedia) Describe() string { return o.Name }

// PutSettings changes values in the project's configuration.
//
// Settings go through a change set like everything else, so that the git
// publisher stages a settings change the same way it stages a post, and so
// that one commit can carry both. A configuration edit deserves the commit,
// the audit record and the undo that a content edit gets.
type PutSettings struct {
	IfRevision Revision
	// Values maps a dotted path to its new value, such as "site.title" or
	// "theme.settings.primary_color". Only the named leaves change: the keys
	// around them, and the comments explaining them, are left alone. A nil
	// value removes its key, so that the default applies again.
	Values map[string]any
}

func (o PutSettings) Kind() OpKind { return OpPutSettings }

func (o PutSettings) Describe() string {
	return strings.Join(slices.Sorted(maps.Keys(o.Values)), ", ")
}

// ChangeTerm renames or removes one term on one item and leaves the rest of
// its file alone. Renaming or removing a term everywhere is a set of these,
// one per item that carries it: a PutContent of each item would also write
// back every value the reader derived for it, such as a timestamp its file
// never declared.
type ChangeTerm struct {
	ID         ID
	IfRevision Revision
	Taxonomy   string
	Term       string
	// To is the term's new name; empty removes the term. An item that already
	// carries To keeps it once.
	To string
}

func (o ChangeTerm) Kind() OpKind { return OpChangeTerm }

func (o ChangeTerm) Describe() string {
	return string(o.ID) + " " + o.Taxonomy + ": " + o.Term + " -> " + o.To
}

var themeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidThemeName reports whether a name can stand for an installed theme: one
// directory under themes/, never a path that leads anywhere else.
func ValidThemeName(name string) bool { return themeName.MatchString(name) }

// PutTheme installs a theme as the directory themes/<Name>.
//
// A theme goes through a change set like anything else the admin writes, so
// the git publisher stages it the way it stages a post, and the commit that
// brought a theme in can be reverted like any other.
type PutTheme struct {
	Name string
	// Files maps each path inside the theme directory, written with forward
	// slashes, to its bytes.
	Files map[string][]byte
	// Replace lets the theme take the place of an installed one of the same
	// name. A file the old one had and the new one lacks is removed.
	Replace bool
}

func (o PutTheme) Kind() OpKind     { return OpPutTheme }
func (o PutTheme) Describe() string { return "themes/" + o.Name }

// DeleteTheme removes an installed theme and everything in its directory.
type DeleteTheme struct {
	Name string
}

func (o DeleteTheme) Kind() OpKind     { return OpDeleteTheme }
func (o DeleteTheme) Describe() string { return "themes/" + o.Name }

// PutPlugin installs a plugin as the directory plugins/<ID>, going through a
// change set for the same reasons a theme does.
type PutPlugin struct {
	ID string
	// Files maps each path inside the plugin directory, written with forward
	// slashes, to its bytes.
	Files map[string][]byte
	// Replace lets it take the place of an installed version. A file the old
	// one had and the new one lacks is removed.
	Replace bool
}

func (o PutPlugin) Kind() OpKind     { return OpPutPlugin }
func (o PutPlugin) Describe() string { return "plugins/" + o.ID }

// DeletePlugin removes an installed plugin and everything in its directory.
type DeletePlugin struct {
	ID string
}

func (o DeletePlugin) Kind() OpKind     { return OpDeletePlugin }
func (o DeletePlugin) Describe() string { return "plugins/" + o.ID }

// DeleteMedia removes a media file.
type DeleteMedia struct {
	Owner ID
	Name  string
}

func (o DeleteMedia) Kind() OpKind     { return OpDeleteMedia }
func (o DeleteMedia) Describe() string { return o.Name }

// ChangeSet is the only unit of change in Kite.
//
// The same structure is simultaneously the git commit, the database
// transaction, the conflict detection unit, the audit record and the undo
// record. Field level mutators are deliberately absent: retrofitting those five
// behaviors onto a wide set of per-field setters is the rewrite this design
// exists to avoid.
type ChangeSet struct {
	Ops     []Op
	Message string
}

// Add appends operations and returns the set, for fluent construction.
func (cs *ChangeSet) Add(ops ...Op) *ChangeSet {
	cs.Ops = append(cs.Ops, ops...)
	return cs
}

// IsEmpty reports whether the set would do nothing.
func (cs *ChangeSet) IsEmpty() bool { return len(cs.Ops) == 0 }

// Result reports what a [Writer] actually did.
type Result struct {
	Revision Revision

	// Written and Removed list every file the store touched, relative to the
	// project root and slash separated. The git publisher stages exactly these
	// paths, which is what lets it commit without disturbing anything else in
	// the user's working tree.
	Written []string
	Removed []string

	// IDs maps each put operation to the ID that was stored, so callers can
	// learn the ID of a newly created item.
	IDs []ID
}

// Writer is the write side of the content store.
type Writer interface {
	Apply(ctx context.Context, cs ChangeSet) (Result, error)
}

// Store is a backend that can both read and write.
type Store interface {
	Reader
	Writer
}
