-- FTS5 virtual table for full-text search
-- Using regular FTS5 table (not external content) for better reliability
-- FTS5 has an implicit rowid column that we'll use to join with links.rowid
DROP TABLE IF EXISTS links_fts;

CREATE VIRTUAL TABLE links_fts USING fts5(
    url,
    title,
    note,
    tags
);

-- Populate FTS5 with existing links
-- FTS5's implicit rowid will match links.rowid
INSERT INTO links_fts(rowid, url, title, note, tags)
SELECT rowid, url, COALESCE(title, ''), COALESCE(note, ''), COALESCE(tags, '')
FROM links;

-- Triggers to keep FTS5 in sync with links table
CREATE TRIGGER IF NOT EXISTS links_ai AFTER INSERT ON links BEGIN
    INSERT INTO links_fts(rowid, url, title, note, tags)
    VALUES (new.rowid, new.url, COALESCE(new.title, ''), COALESCE(new.note, ''), COALESCE(new.tags, ''));
END;

CREATE TRIGGER IF NOT EXISTS links_ad AFTER DELETE ON links BEGIN
    DELETE FROM links_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS links_au AFTER UPDATE ON links BEGIN
    DELETE FROM links_fts WHERE rowid = old.rowid;
    INSERT INTO links_fts(rowid, url, title, note, tags)
    VALUES (new.rowid, new.url, COALESCE(new.title, ''), COALESCE(new.note, ''), COALESCE(new.tags, ''));
END;

