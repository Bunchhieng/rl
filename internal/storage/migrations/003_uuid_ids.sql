-- Migration to convert integer IDs to short UUID strings
-- This migration handles existing databases with INTEGER IDs
-- For new databases, migration 001 already creates TEXT IDs

-- Step 1: Check if we need to migrate (id is INTEGER type)
-- SQLite stores INTEGER PRIMARY KEY as integer type
-- We detect this by checking if we can cast id to integer

-- Step 2: Create new table with TEXT ID
CREATE TABLE IF NOT EXISTS links_new (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    title TEXT,
    note TEXT,
    tags TEXT,
    created_at TEXT NOT NULL,
    read_at TEXT
);

-- Step 3: Copy data, generating short IDs for integer IDs
-- Use hex encoding of random bytes as a simple approach (SQLite has hex() function)
-- We'll generate 12-character IDs using hex(randomblob(6)) which gives us 12 hex chars
INSERT INTO links_new (id, url, title, note, tags, created_at, read_at)
SELECT 
    CASE 
        WHEN typeof(id) = 'integer' THEN
            -- Generate short ID: use hex of random bytes, take first 12 chars
            -- This gives us 12 hex characters = 48 bits of entropy
            substr(lower(hex(randomblob(6))), 1, 12)
        ELSE
            -- Keep existing TEXT IDs (already migrated or new database)
            id
    END as id,
    url,
    title,
    note,
    tags,
    created_at,
    read_at
FROM links;

-- Step 4: Drop old table and rename new one
DROP TABLE IF EXISTS links;
ALTER TABLE links_new RENAME TO links;

-- Step 5: Recreate indexes
CREATE INDEX IF NOT EXISTS idx_links_read_at ON links(read_at);
CREATE INDEX IF NOT EXISTS idx_links_created_at ON links(created_at);
CREATE INDEX IF NOT EXISTS idx_links_tags ON links(tags);

-- Note: FTS5 table will be recreated by migration 002 if needed

