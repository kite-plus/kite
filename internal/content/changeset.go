package content

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned when the target of an operation does not exist.
	ErrNotFound = errors.New("content: not found")

	// ErrConflict is returned when IfRevision does not match what is stored,
	// meaning the item changed underneath the caller. The API surfaces it as
	// 409 together with a three-way diff.
	ErrConflict = errors.New("content: revision conflict")

	// ErrDuplicateID is returned when two items claim the same ID. Copying a
	// bundle directory is a normal user action, so this is always reported and
	// never silently repaired.
	ErrDuplicateID = errors.New("content: duplicate id")
)

// ConflictError carries the data the admin needs to render a three-way merge.
type ConflictError struct {
	ID       ID
	Expected Revision
	Actual   Revision
	Base     []byte
	Ours     []byte
	Theirs   []byte
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("content %s: revision conflict (expected %s, found %s)", e.ID, e.Expected, e.Actual)
}

func (e *ConflictError) Unwrap() error { return ErrConflict }

// OpKind names the kind of an [Op].
type OpKind string

const (
	OpPutContent    OpKind = "put_content"
	OpDeleteContent OpKind = "delete_content"
	OpMoveContent   OpKind = "move_content"
	OpPutMedia      OpKind = "put_media"
	OpDeleteMedia   OpKind = "delete_media"
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

// PutMedia writes a media file, usually into the owning item's bundle.
type PutMedia struct {
	Owner ID
	Name  string
	Data  []byte
}

func (o PutMedia) Kind() OpKind     { return OpPutMedia }
func (o PutMedia) Describe() string { return o.Name }

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
// behaviours onto a wide set of per-field setters is the rewrite this design
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
