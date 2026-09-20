package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/kite-plus/kite/internal/store/file"
)

// racyWindow mirrors git's "racy timestamp" rule.
//
// A filesystem whose timestamps have coarse granularity allows a file to be
// modified, indexed and modified again inside a single tick, leaving stat data
// that matches while the contents differ. Any file whose mtime falls inside
// this window of the last index write is therefore rehashed regardless of what
// stat says.
const racyWindow = time.Second

// Stats reports what a reconcile did.
type Stats struct {
	Scanned  int `json:"scanned"`
	Indexed  int `json:"indexed"`
	Skipped  int `json:"skipped"`
	Removed  int `json:"removed"`
	Problems int `json:"problems"`

	Duration time.Duration `json:"-"`
}

type fileRow struct {
	size        int64
	mtimeNS     int64
	hash        string
	indexedAtNS int64
}

// Reconcile brings the index into agreement with the files on disk.
//
// The walk is stat only; a file is re-read just when its size or mtime differs
// from the recorded values, or when the racy timestamp rule applies, or when
// git reports that the checkout moved it.
func (ix *Index) Reconcile(ctx context.Context) (Stats, error) {
	start := time.Now()
	var stats Stats

	known, err := ix.knownFiles(ctx)
	if err != nil {
		return stats, err
	}
	forced, err := ix.pathsChangedByGit(ctx)
	if err != nil {
		return stats, err
	}

	var problems []file.Problem
	seen := make(map[string]struct{}, len(known))
	now := time.Now().UnixNano()

	tx, err := ix.db.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback() //nolint:errcheck // committed below on the happy path

	walkErr := ix.scanner.Walk(func(st file.Stat) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		stats.Scanned++
		seen[st.Path] = struct{}{}

		if _, mustRead := forced[st.Path]; !mustRead {
			if prev, ok := known[st.Path]; ok && !needsReread(prev, st) {
				stats.Skipped++
				return nil
			}
		}

		entry, err := ix.scanner.Load(st)
		if err != nil {
			problems = append(problems, file.Problem{Path: st.Path, Err: err})
			return nil
		}
		if entry.Item.ID == "" {
			problems = append(problems, file.Problem{
				Path: st.Path,
				Err:  errors.New("no id in front matter; run 'kite doctor --fix-ids'"),
			})
			return nil
		}
		if err := upsert(ctx, tx, entry, now); err != nil {
			return err
		}
		stats.Indexed++
		return nil
	})
	if walkErr != nil {
		return stats, walkErr
	}

	for _, path := range slices.Sorted(maps.Keys(known)) {
		if _, ok := seen[path]; ok {
			continue
		}
		if err := deleteByPath(ctx, tx, path); err != nil {
			return stats, err
		}
		stats.Removed++
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("index: commit: %w", err)
	}
	if err := ix.recordGitHead(ctx); err != nil {
		return stats, err
	}

	stats.Problems = len(problems)
	stats.Duration = time.Since(start)
	if len(problems) > 0 {
		errs := make([]error, 0, len(problems))
		for _, p := range problems {
			errs = append(errs, p)
		}
		return stats, errors.Join(errs...)
	}
	return stats, nil
}

// Rebuild discards every row and indexes from scratch. The result must be
// identical to an incremental reconcile; `kite index --verify` relies on that.
func (ix *Index) Rebuild(ctx context.Context) (Stats, error) {
	for _, stmt := range []string{`DELETE FROM terms`, `DELETE FROM contents`, `DELETE FROM files`} {
		if _, err := ix.db.ExecContext(ctx, stmt); err != nil {
			return Stats{}, fmt.Errorf("index: clear: %w", err)
		}
	}
	if err := ix.setMeta(metaGitHead, ""); err != nil {
		return Stats{}, err
	}
	return ix.Reconcile(ctx)
}

func needsReread(prev fileRow, st file.Stat) bool {
	if prev.size != st.Size || prev.mtimeNS != st.ModTime {
		return true
	}
	// Racy timestamp: the file was touched close enough to the last index
	// write that identical stat data proves nothing.
	return st.ModTime >= prev.indexedAtNS-int64(racyWindow)
}

func (ix *Index) knownFiles(ctx context.Context) (map[string]fileRow, error) {
	rows, err := ix.db.QueryContext(ctx, `SELECT path, size, mtime_ns, content_sha256, indexed_at_ns FROM files`)
	if err != nil {
		return nil, fmt.Errorf("index: read file table: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]fileRow)
	for rows.Next() {
		var path string
		var r fileRow
		if err := rows.Scan(&path, &r.size, &r.mtimeNS, &r.hash, &r.indexedAtNS); err != nil {
			return nil, err
		}
		out[path] = r
	}
	return out, rows.Err()
}

func upsert(ctx context.Context, tx *sql.Tx, e *file.Entry, nowNS int64) error {
	item := e.Item

	meta, err := json.Marshal(orEmptyMap(item.Meta))
	if err != nil {
		return err
	}
	aliases, err := json.Marshal(orEmptySlice(item.Aliases))
	if err != nil {
		return err
	}

	// A file may have been renamed onto an ID that already sits at another
	// path; clearing both keys first keeps the unique indexes satisfied.
	if _, err := tx.ExecContext(ctx, `DELETE FROM contents WHERE path = ? OR id = ?`, e.Path, string(item.ID)); err != nil {
		return fmt.Errorf("index: replace %s: %w", e.Path, err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO contents (
			id, kind, slug, title, status, locale, locator, path, revision,
			body, body_format, excerpt, meta_json, aliases_json,
			created_at, updated_at, published_at, deleted_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		string(item.ID), string(item.Kind), item.Slug, item.Title, string(item.Status),
		item.Locale, string(item.Locator), e.Path, string(item.Revision),
		item.Body.Raw, string(item.Body.Format), summarize(description(item.Meta), item.Body.Raw),
		string(meta), string(aliases),
		unixOrZero(item.CreatedAt), unixOrZero(item.UpdatedAt),
		unixPtr(item.PublishedAt), unixPtr(item.DeletedAt),
	)
	if err != nil {
		return fmt.Errorf("index: insert %s: %w", e.Path, err)
	}

	for _, taxonomy := range slices.Sorted(maps.Keys(item.Taxonomies)) {
		for i, term := range item.Taxonomies[taxonomy] {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR REPLACE INTO terms (content_id, taxonomy, term, position) VALUES (?,?,?,?)`,
				string(item.ID), taxonomy, term, i); err != nil {
				return fmt.Errorf("index: insert term %s/%s: %w", taxonomy, term, err)
			}
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO files (path, size, mtime_ns, content_sha256, indexed_at_ns)
		VALUES (?,?,?,?,?)
		ON CONFLICT (path) DO UPDATE SET
			size = excluded.size, mtime_ns = excluded.mtime_ns,
			content_sha256 = excluded.content_sha256, indexed_at_ns = excluded.indexed_at_ns`,
		e.Path, e.Size, e.ModTime, string(e.Hash), nowNS)
	if err != nil {
		return fmt.Errorf("index: record file %s: %w", e.Path, err)
	}
	return nil
}

func deleteByPath(ctx context.Context, tx *sql.Tx, path string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM contents WHERE path = ?`, path); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM files WHERE path = ?`, path)
	return err
}

// description returns an author-written summary from the item's metadata.
func description(meta map[string]any) string {
	if v, ok := meta["description"].(string); ok {
		return v
	}
	return ""
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func unixPtr(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Unix()
}

func orEmptyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func orEmptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
