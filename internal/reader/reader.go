// Package reader implements the one and only content.Reader.
//
// It queries the read model schema defined in internal/index. In file mode an
// indexer derives those rows from markdown; in database mode a writer fills
// them in directly. Because both modes share this single implementation, a
// theme cannot observe which one it is running under.
package reader

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/content"
)

// Reader queries the read model.
type Reader struct {
	db *sql.DB
}

// New returns a reader over an open read model database.
func New(db *sql.DB) *Reader { return &Reader{db: db} }

// Caps reports what this reader supports.
func (r *Reader) Caps() content.Capabilities {
	return content.Capabilities{FullText: true, Aggregate: true}
}

const selectColumns = `
	id, kind, slug, title, status, locale, locator, path, revision,
	body, body_format, excerpt, meta_json, aliases_json,
	created_at, updated_at, published_at, deleted_at`

// Get returns one item by ID.
func (r *Reader) Get(ctx context.Context, id content.ID) (*content.Content, error) {
	row := r.db.QueryRowContext(ctx, `SELECT`+selectColumns+` FROM contents WHERE id = ?`, string(id))
	item, err := scanContent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", content.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	return r.withTerms(ctx, item)
}

// GetMany returns many items by ID, in the order given.
func (r *Reader) GetMany(ctx context.Context, ids []content.ID) ([]*content.Content, error) {
	byID := make(map[content.ID]*content.Content, len(ids))
	for chunk := range slices.Chunk(ids, content.MaxLimit) {
		rows, err := r.db.QueryContext(ctx,
			`SELECT`+selectColumns+` FROM contents WHERE id IN (`+placeholders(len(chunk))+`)`, toAny(chunk)...)
		if err != nil {
			return nil, fmt.Errorf("reader: load: %w", err)
		}
		var found []string
		for rows.Next() {
			item, err := scanContent(rows)
			if err != nil {
				_ = rows.Close()
				return nil, err
			}
			byID[item.ID] = item
			found = append(found, string(item.ID))
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}

		terms, err := r.termsFor(ctx, found)
		if err != nil {
			return nil, err
		}
		for _, id := range found {
			byID[content.ID(id)].Taxonomies = terms[id]
		}
	}

	out := make([]*content.Content, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

// GetBySlug returns one item by kind and slug.
func (r *Reader) GetBySlug(ctx context.Context, kind content.Kind, slug string) (*content.Content, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT`+selectColumns+` FROM contents WHERE kind = ? AND slug = ? ORDER BY locale LIMIT 1`,
		string(kind), slug)
	item, err := scanContent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s/%s", content.ErrNotFound, kind, slug)
	}
	if err != nil {
		return nil, err
	}
	return r.withTerms(ctx, item)
}

// Query returns a page of summaries.
func (r *Reader) Query(ctx context.Context, q content.Query) (content.Page[content.Summary], error) {
	var page content.Page[content.Summary]
	if err := q.Normalize(); err != nil {
		return page, err
	}

	where, args, err := buildWhere(q)
	if err != nil {
		return page, err
	}
	order := buildOrder(q.Sort)

	// One extra row tells us whether another page exists without a count(*).
	sqlText := `SELECT id, kind, slug, title, status, locale, locator, revision, excerpt,
		created_at, updated_at, published_at
		FROM contents` + where + order + ` LIMIT ?`
	rows, err := r.db.QueryContext(ctx, sqlText, append(args, q.Limit+1)...)
	if err != nil {
		return page, fmt.Errorf("reader: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var s content.Summary
		var createdAt, updatedAt int64
		var publishedAt sql.NullInt64
		if err := rows.Scan(&s.ID, &s.Kind, &s.Slug, &s.Title, &s.Status, &s.Locale,
			&s.Locator, &s.Revision, &s.Excerpt, &createdAt, &updatedAt, &publishedAt); err != nil {
			return page, err
		}
		s.CreatedAt = fromUnix(createdAt)
		s.UpdatedAt = fromUnix(updatedAt)
		s.PublishedAt = fromNullUnix(publishedAt)
		page.Items = append(page.Items, s)
		ids = append(ids, string(s.ID))
	}
	if err := rows.Err(); err != nil {
		return page, err
	}

	if len(page.Items) > q.Limit {
		page.Items = page.Items[:q.Limit]
		ids = ids[:q.Limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = cursorFor(last, q.Sort).Encode()
	}

	terms, err := r.termsFor(ctx, ids)
	if err != nil {
		return page, err
	}
	for i := range page.Items {
		page.Items[i].Taxonomies = terms[string(page.Items[i].ID)]
	}
	return page, nil
}

// Count reports how many items a query selects.
//
// It shares buildWhere with Query, so a total can never describe a different
// set than the rows it is a total of.
func (r *Reader) Count(ctx context.Context, q content.Query) (int, error) {
	if err := q.Normalize(); err != nil {
		return 0, err
	}
	q.Cursor = "" // how far paging has got is not part of the question

	where, args, err := buildWhere(q)
	if err != nil {
		return 0, err
	}
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM contents`+where, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("reader: count: %w", err)
	}
	return n, nil
}

// CountTerms aggregates term usage across the items a query selects.
func (r *Reader) CountTerms(ctx context.Context, taxonomy string, q content.Query) ([]content.TermCount, error) {
	if err := q.Normalize(); err != nil {
		return nil, err
	}
	q.Cursor = "" // an aggregate is over the whole set, not one page of it

	where, args, err := buildWhere(q)
	if err != nil {
		return nil, err
	}

	sqlText := `SELECT t.term, count(*) FROM terms t
		JOIN contents c ON c.id = t.content_id
		WHERE t.taxonomy = ? AND c.id IN (SELECT id FROM contents` + where + `)
		GROUP BY t.term ORDER BY count(*) DESC, t.term ASC`
	rows, err := r.db.QueryContext(ctx, sqlText, append([]any{taxonomy}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("reader: count terms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []content.TermCount
	for rows.Next() {
		tc := content.TermCount{Taxonomy: taxonomy}
		if err := rows.Scan(&tc.Term, &tc.Count); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanContent(row scanner) (*content.Content, error) {
	var (
		item                  content.Content
		metaJSON, aliasesJSON string
		bodyFormat            string
		createdAt, updatedAt  int64
		publishedAt, deleted  sql.NullInt64
		excerpt               string
		path                  string
	)
	err := row.Scan(&item.ID, &item.Kind, &item.Slug, &item.Title, &item.Status,
		&item.Locale, &item.Locator, &path, &item.Revision,
		&item.Body.Raw, &bodyFormat, &excerpt, &metaJSON, &aliasesJSON,
		&createdAt, &updatedAt, &publishedAt, &deleted)
	if err != nil {
		return nil, err
	}

	item.Body.Format = content.Format(bodyFormat)
	item.CreatedAt = fromUnix(createdAt)
	item.UpdatedAt = fromUnix(updatedAt)
	item.PublishedAt = fromNullUnix(publishedAt)
	item.DeletedAt = fromNullUnix(deleted)

	if err := json.Unmarshal([]byte(metaJSON), &item.Meta); err != nil {
		return nil, fmt.Errorf("reader: decode meta of %s: %w", item.ID, err)
	}
	if err := json.Unmarshal([]byte(aliasesJSON), &item.Aliases); err != nil {
		return nil, fmt.Errorf("reader: decode aliases of %s: %w", item.ID, err)
	}
	return &item, nil
}

func (r *Reader) withTerms(ctx context.Context, item *content.Content) (*content.Content, error) {
	terms, err := r.termsFor(ctx, []string{string(item.ID)})
	if err != nil {
		return nil, err
	}
	item.Taxonomies = terms[string(item.ID)]
	return item, nil
}

func (r *Reader) termsFor(ctx context.Context, ids []string) (map[string]map[string][]string, error) {
	out := make(map[string]map[string][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	sqlText := `SELECT content_id, taxonomy, term FROM terms
		WHERE content_id IN (` + placeholders(len(ids)) + `)
		ORDER BY content_id, taxonomy, position`
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("reader: load terms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, taxonomy, term string
		if err := rows.Scan(&id, &taxonomy, &term); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = make(map[string][]string)
		}
		out[id][taxonomy] = append(out[id][taxonomy], term)
	}
	return out, rows.Err()
}

func buildWhere(q content.Query) (string, []any, error) {
	var clauses []string
	var args []any

	add := func(clause string, values ...any) {
		clauses = append(clauses, clause)
		args = append(args, values...)
	}

	if len(q.Kinds) > 0 {
		add(`kind IN (`+placeholders(len(q.Kinds))+`)`, toAny(q.Kinds)...)
	}
	if len(q.Statuses) > 0 {
		add(`status IN (`+placeholders(len(q.Statuses))+`)`, toAny(q.Statuses)...)
	}
	if len(q.Locales) > 0 {
		add(`locale IN (`+placeholders(len(q.Locales))+`)`, toAny(q.Locales)...)
	}
	if len(q.IDs) > 0 {
		add(`id IN (`+placeholders(len(q.IDs))+`)`, toAny(q.IDs)...)
	}
	if !q.IncludeDeleted {
		add(`deleted_at IS NULL`)
	}
	if q.PublicAt != nil {
		// A scheduled item without a date never becomes public, and a NULL
		// comparison is false, which is the same answer.
		add(`(deleted_at IS NULL AND (status = ? OR (status = ? AND published_at <= ?)))`,
			string(content.StatusPublished), string(content.StatusScheduled), q.PublicAt.Unix())
	}
	if q.Published != nil {
		if q.Published.From != nil {
			add(`published_at >= ?`, q.Published.From.Unix())
		}
		if q.Published.To != nil {
			add(`published_at <= ?`, q.Published.To.Unix())
		}
	}
	if q.Updated != nil {
		if q.Updated.From != nil {
			add(`updated_at >= ?`, q.Updated.From.Unix())
		}
		if q.Updated.To != nil {
			add(`updated_at <= ?`, q.Updated.To.Unix())
		}
	}
	if q.Text != "" {
		like := "%" + q.Text + "%"
		add(`(title LIKE ? OR excerpt LIKE ? OR body LIKE ?)`, like, like, like)
	}

	for taxonomy, terms := range q.TermsAny {
		if len(terms) == 0 {
			continue
		}
		add(`id IN (SELECT content_id FROM terms WHERE taxonomy = ? AND term IN (`+
			placeholders(len(terms))+`))`, append([]any{taxonomy}, toAny(terms)...)...)
	}
	for taxonomy, terms := range q.TermsAll {
		for _, term := range terms {
			add(`id IN (SELECT content_id FROM terms WHERE taxonomy = ? AND term = ?)`, taxonomy, term)
		}
	}

	if q.Cursor != "" {
		cursor, err := content.DecodeCursor(q.Cursor)
		if err != nil {
			return "", nil, err
		}
		clause, cursorArgs, err := cursorClause(q.Sort, cursor)
		if err != nil {
			return "", nil, err
		}
		add(clause, cursorArgs...)
	}

	if len(clauses) == 0 {
		return "", args, nil
	}
	return ` WHERE ` + strings.Join(clauses, " AND "), args, nil
}

// cursorClause turns a cursor into the lexicographic comparison that continues
// the ordering. With sort keys (a, b) it expands to
// "a < ?a OR (a = ?a AND b < ?b)", which a range scan over the matching index
// can satisfy without sorting.
func cursorClause(sort []content.SortKey, cursor content.Cursor) (string, []any, error) {
	if len(cursor.Values) != len(sort) {
		return "", nil, fmt.Errorf("%w: cursor does not match the sort order", content.ErrUnsupportedQuery)
	}

	var branches []string
	var args []any
	for i := range sort {
		var parts []string
		for j := 0; j < i; j++ {
			parts = append(parts, sortColumn(sort[j].Field)+" = ?")
			args = append(args, cursorArg(sort[j].Field, cursor.Values[j]))
		}
		op := ">"
		if sort[i].Desc {
			op = "<"
		}
		parts = append(parts, sortColumn(sort[i].Field)+" "+op+" ?")
		args = append(args, cursorArg(sort[i].Field, cursor.Values[i]))
		branches = append(branches, "("+strings.Join(parts, " AND ")+")")
	}
	return "(" + strings.Join(branches, " OR ") + ")", args, nil
}

func buildOrder(sort []content.SortKey) string {
	parts := make([]string, 0, len(sort))
	for _, k := range sort {
		dir := " ASC"
		if k.Desc {
			dir = " DESC"
		}
		parts = append(parts, sortColumn(k.Field)+dir)
	}
	return ` ORDER BY ` + strings.Join(parts, ", ")
}

// sortColumn maps a sort field to a column. Query.Normalize has already
// rejected unknown fields, so no user input reaches the SQL text here.
func sortColumn(field string) string {
	switch field {
	case content.SortPublishedAt:
		// Items without a publication date sort as oldest rather than first.
		return `COALESCE(published_at, 0)`
	case content.SortUpdatedAt:
		return `updated_at`
	case content.SortCreatedAt:
		return `created_at`
	case content.SortTitle:
		return `title`
	case content.SortSlug:
		return `slug`
	default:
		return `id`
	}
}

func cursorFor(s content.Summary, sort []content.SortKey) content.Cursor {
	values := make([]string, 0, len(sort))
	for _, k := range sort {
		switch k.Field {
		case content.SortPublishedAt:
			values = append(values, strconv.FormatInt(unixOrZero(s.PublishedAt), 10))
		case content.SortUpdatedAt:
			values = append(values, strconv.FormatInt(s.UpdatedAt.Unix(), 10))
		case content.SortCreatedAt:
			values = append(values, strconv.FormatInt(s.CreatedAt.Unix(), 10))
		case content.SortTitle:
			values = append(values, s.Title)
		case content.SortSlug:
			values = append(values, s.Slug)
		default:
			values = append(values, string(s.ID))
		}
	}
	return content.Cursor{Values: values}
}

func cursorArg(field, value string) any {
	switch field {
	case content.SortPublishedAt, content.SortUpdatedAt, content.SortCreatedAt:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return value
	}
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func toAny[T ~string](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}

func fromUnix(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

func fromNullUnix(v sql.NullInt64) *time.Time {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	t := time.Unix(v.Int64, 0).UTC()
	return &t
}

func unixOrZero(t *time.Time) int64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return t.Unix()
}

// Reader must satisfy the domain contract.
var _ content.Reader = (*Reader)(nil)
