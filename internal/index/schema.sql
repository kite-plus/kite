-- The single read model.
--
-- In file mode these rows are derived from markdown files by the indexer; in
-- database mode the writer populates them directly. Either way every reader,
-- every theme and every API response sees the same shape, which is what stops
-- static and dynamic rendering from drifting apart.
--
-- In file mode this whole database is a cache: deleting it and running
-- `kite index` must reproduce it exactly. Nothing may live here that cannot be
-- derived from content/, the config and the theme.

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- files tracks the stat and hash of every source file so a reconcile can skip
-- unchanged ones. The columns mirror what git caches, for the same reason.
CREATE TABLE IF NOT EXISTS files (
    path           TEXT PRIMARY KEY,
    size           INTEGER NOT NULL,
    mtime_ns       INTEGER NOT NULL,
    content_sha256 TEXT    NOT NULL,
    indexed_at_ns  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS contents (
    id           TEXT PRIMARY KEY,
    kind         TEXT    NOT NULL,
    slug         TEXT    NOT NULL,
    title        TEXT    NOT NULL,
    status       TEXT    NOT NULL,
    locale       TEXT    NOT NULL DEFAULT '',
    locator      TEXT    NOT NULL,
    path         TEXT    NOT NULL,
    revision     TEXT    NOT NULL,
    body         TEXT    NOT NULL DEFAULT '',
    body_format  TEXT    NOT NULL DEFAULT 'markdown',
    excerpt      TEXT    NOT NULL DEFAULT '',
    meta_json    TEXT    NOT NULL DEFAULT '{}',
    aliases_json TEXT    NOT NULL DEFAULT '[]',
    created_at   INTEGER NOT NULL DEFAULT 0,
    updated_at   INTEGER NOT NULL DEFAULT 0,
    published_at INTEGER,
    deleted_at   INTEGER
);

CREATE UNIQUE INDEX IF NOT EXISTS contents_path      ON contents (path);
CREATE UNIQUE INDEX IF NOT EXISTS contents_kind_slug ON contents (kind, slug, locale);

-- The default ordering is (published_at DESC, id DESC); the index exists so
-- that cursor pagination over it is a range scan rather than a sort.
CREATE INDEX IF NOT EXISTS contents_default_order ON contents (published_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS contents_kind_status    ON contents (kind, status);
CREATE INDEX IF NOT EXISTS contents_updated        ON contents (updated_at DESC, id DESC);

-- terms is a materialized projection of the string lists carried on each item,
-- never a source of truth. Renaming a term means rewriting the items that use
-- it, in file mode and database mode alike.
CREATE TABLE IF NOT EXISTS terms (
    content_id TEXT    NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    taxonomy   TEXT    NOT NULL,
    term       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (content_id, taxonomy, term)
);

CREATE INDEX IF NOT EXISTS terms_lookup ON terms (taxonomy, term);
