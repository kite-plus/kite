// Package index maintains the derived read model.
//
// In file mode the markdown files are the source of truth and this database is
// a cache: deleting it and reindexing must reproduce it exactly. Data flows one
// way only, from files into the index, never back.
package index

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite" // pure Go driver: keeps cross compilation cgo-free

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/store/file"
)

//go:embed schema.sql
var schemaSQL string

// SchemaVersion is bumped whenever the shape of the read model changes. A
// mismatch discards the database rather than migrating it, which is safe
// precisely because nothing here is a source of truth.
const SchemaVersion = 1

// Path is where the derived index lives, relative to the project root.
const Path = ".kite/cache/index.db"

const (
	metaSchemaVersion = "schema_version"
	metaGitHead       = "git_head"
)

// Index is the derived read model together with the machinery that keeps it
// coherent with the files it was derived from.
type Index struct {
	db      *sql.DB
	root    string
	types   *content.Registry
	scanner *file.Scanner
}

// Open opens or creates the index for a project. A database written by a
// different schema version is discarded and rebuilt.
func Open(root string, types *content.Registry) (*Index, error) {
	dbPath := filepath.Join(root, filepath.FromSlash(Path))
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	ix, err := open(dbPath, root, types)
	if err != nil {
		return nil, err
	}

	// The version must be read before it is written, or a database left by an
	// older build would be stamped as current and silently kept.
	version, found, err := ix.schemaVersion()
	if err != nil {
		ix.Close()
		return nil, err
	}
	if found && version != SchemaVersion {
		if err := ix.reset(dbPath); err != nil {
			return nil, err
		}
	}
	if err := ix.setMeta(metaSchemaVersion, strconv.Itoa(SchemaVersion)); err != nil {
		ix.Close()
		return nil, err
	}
	return ix, nil
}

// OpenMemory opens an index that lives only in memory, for tests.
func OpenMemory(root string, types *content.Registry) (*Index, error) {
	ix, err := open(":memory:", root, types)
	if err != nil {
		return nil, err
	}
	if err := ix.setMeta(metaSchemaVersion, strconv.Itoa(SchemaVersion)); err != nil {
		ix.Close()
		return nil, err
	}
	return ix, nil
}

func open(dsn, root string, types *content.Registry) (*Index, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("index: open %s: %w", dsn, err)
	}
	// SQLite tolerates a single writer; serialising here avoids SQLITE_BUSY
	// entirely rather than retrying around it.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("index: apply schema: %w", err)
	}
	return &Index{db: db, root: root, types: types, scanner: file.NewScanner(root, types)}, nil
}

// reset discards the database and starts over.
func (ix *Index) reset(dbPath string) error {
	if err := ix.db.Close(); err != nil {
		return err
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(dbPath + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("index: discard stale database: %w", err)
		}
	}
	fresh, err := open(dbPath, ix.root, ix.types)
	if err != nil {
		return err
	}
	*ix = *fresh
	return nil
}

// Close releases the database.
func (ix *Index) Close() error { return ix.db.Close() }

// DB exposes the underlying handle so that the reader can query the same
// schema a SQL-backed store would populate.
func (ix *Index) DB() *sql.DB { return ix.db }

func (ix *Index) setMeta(key, value string) error {
	_, err := ix.db.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("index: write meta %s: %w", key, err)
	}
	return nil
}

func (ix *Index) meta(key string) (string, error) {
	var v string
	err := ix.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("index: read meta %s: %w", key, err)
	}
	return v, nil
}

// schemaVersion reports the recorded version and whether one was recorded at
// all, so that a fresh database is not mistaken for a stale one.
func (ix *Index) schemaVersion() (version int, found bool, err error) {
	v, err := ix.meta(metaSchemaVersion)
	if err != nil || v == "" {
		return 0, false, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		// An unreadable version is treated as stale, which rebuilds.
		return -1, true, nil
	}
	return n, true, nil
}

// Count returns how many items the index holds.
func (ix *Index) Count(ctx context.Context) (int, error) {
	var n int
	err := ix.db.QueryRowContext(ctx, `SELECT count(*) FROM contents`).Scan(&n)
	return n, err
}
