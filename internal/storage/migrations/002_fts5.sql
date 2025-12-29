-- FTS5 virtual table for full-text search
-- Using external content - FTS5 reads directly from links table
DROP TABLE IF EXISTS links_fts;

CREATE VIRTUAL TABLE links_fts USING fts5(
    url,
    title,
    note,
    tags,
    content='links',
    content_rowid='rowid'
);

-- Triggers to notify FTS5 of changes (required for external content)
CREATE TRIGGER IF NOT EXISTS links_ai AFTER INSERT ON links BEGIN
    INSERT INTO links_fts(links_fts) VALUES('rebuild');
END;

CREATE TRIGGER IF NOT EXISTS links_ad AFTER DELETE ON links BEGIN
    INSERT INTO links_fts(links_fts) VALUES('rebuild');
END;

CREATE TRIGGER IF NOT EXISTS links_au AFTER UPDATE ON links BEGIN
    INSERT INTO links_fts(links_fts) VALUES('rebuild');
END;

