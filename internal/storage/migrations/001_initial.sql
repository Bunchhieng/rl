-- Initial schema for read later links
-- Note: For new databases, we use TEXT ID (short UUID)
-- Migration 003 will convert existing INTEGER IDs to TEXT

CREATE TABLE IF NOT EXISTS links (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    title TEXT,
    note TEXT,
    tags TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    read_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_links_read_at ON links(read_at);
CREATE INDEX IF NOT EXISTS idx_links_created_at ON links(created_at);
CREATE INDEX IF NOT EXISTS idx_links_tags ON links(tags);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

