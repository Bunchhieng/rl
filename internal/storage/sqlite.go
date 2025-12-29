package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bunchhieng/rl/internal/model"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements Storage using SQLite.
type SQLiteStorage struct {
	db *sqlx.DB
}

// NewSQLiteStorage creates a new SQLite storage instance.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	var dsn string
	if dbPath == ":memory:" {
		dsn = dbPath + "?_pragma=journal_mode(DELETE)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"
	} else {
		dsn = dbPath + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	}

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	storage := &SQLiteStorage{db: db}
	ctx := context.Background()
	if err := runMigrations(ctx, db.DB); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return storage, nil
}

type linkRow struct {
	ID        string         `db:"id"`
	URL       string         `db:"url"`
	Title     sql.NullString `db:"title"`
	Note      sql.NullString `db:"note"`
	Tags      sql.NullString `db:"tags"`
	CreatedAt string         `db:"created_at"`
	ReadAt    sql.NullString `db:"read_at"`
}

func (r *linkRow) toLink() *model.Link {
	link := &model.Link{
		ID:  r.ID,
		URL: r.URL,
	}
	if r.Title.Valid {
		link.Title = r.Title.String
	}
	if r.Note.Valid {
		link.Note = r.Note.String
	}
	if r.Tags.Valid {
		link.Tags = r.Tags.String
	}
	link.CreatedAt = parseSQLiteTime(r.CreatedAt)
	if r.ReadAt.Valid && r.ReadAt.String != "" {
		readAt := parseSQLiteTime(r.ReadAt.String)
		link.ReadAt = &readAt
	}
	return link
}

// Add creates a new link or updates an existing one.
func (s *SQLiteStorage) Add(ctx context.Context, link *model.Link) (*model.Link, error) {
	if err := link.Validate(); err != nil {
		return nil, err
	}

	// Check if link already exists
	var existing linkRow
	err := s.db.GetContext(ctx, &existing,
		"SELECT id, url, title, note, tags, created_at, read_at FROM links WHERE url = ?", link.URL)

	if err == nil {
		// Link exists - update it
		existingLink := existing.toLink()

		// Merge: preserve existing title/note if present, merge tags
		newTitle := existingLink.Title
		if link.Title != "" {
			newTitle = link.Title
		}
		newNote := existingLink.Note
		if link.Note != "" {
			newNote = link.Note
		}

		// Merge tags
		if link.Tags != "" {
			mergeLink := &model.Link{Tags: link.Tags}
			existingLink.MergeTags(mergeLink)
		}
		newTags := existingLink.Tags

		// Use DELETE + INSERT to avoid driver issues with UPDATE
		_, err = s.db.ExecContext(ctx, "DELETE FROM links WHERE url = ?", link.URL)
		if err != nil {
			return nil, fmt.Errorf("delete existing link: %w", err)
		}

		// Re-insert with merged data, preserving original created_at
		existingCreatedAt := existingLink.CreatedAt.Format(time.RFC3339)
		if existingLink.CreatedAt.IsZero() {
			existingCreatedAt = time.Now().Format(time.RFC3339)
		}

		var readAt sql.NullString
		if existingLink.ReadAt != nil {
			readAt.String = existingLink.ReadAt.Format(time.RFC3339)
			readAt.Valid = true
		}

		_, err = s.db.ExecContext(ctx,
			"INSERT INTO links (id, url, title, note, tags, created_at, read_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			existingLink.ID, link.URL, newTitle, newNote, newTags, existingCreatedAt, readAt)
		if err != nil {
			return nil, fmt.Errorf("re-insert updated link: %w", err)
		}

		// Get the updated link by ID (preserved from existing link)
		return s.Get(ctx, existingLink.ID)
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check existing link: %w", err)
	}

	// err == sql.ErrNoRows, so link doesn't exist - insert new one

	// Link doesn't exist - insert new one
	// Generate short ID for new link
	link.ID = model.GenerateShortID()

	var readAt sql.NullString
	if link.ReadAt != nil {
		readAt.String = link.ReadAt.Format(time.RFC3339)
		readAt.Valid = true
	}

	createdAtStr := link.CreatedAt.Format(time.RFC3339)
	if link.CreatedAt.IsZero() {
		createdAtStr = time.Now().Format(time.RFC3339)
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO links (id, url, title, note, tags, created_at, read_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		link.ID, link.URL, link.Title, link.Note, link.Tags, createdAtStr, readAt)
	if err != nil {
		return nil, fmt.Errorf("insert link: %w", err)
	}

	createdAtTime := parseSQLiteTime(createdAtStr)
	result := &model.Link{
		ID:        link.ID,
		URL:       link.URL,
		Title:     link.Title,
		Note:      link.Note,
		Tags:      link.Tags,
		CreatedAt: createdAtTime,
		ReadAt:    link.ReadAt,
	}

	return result, nil
}

// Get retrieves a link by ID.
func (s *SQLiteStorage) Get(ctx context.Context, id string) (*model.Link, error) {
	if !model.ValidateShortID(id) {
		return nil, fmt.Errorf("invalid ID format")
	}
	var row linkRow
	err := s.db.GetContext(ctx, &row,
		"SELECT id, url, title, note, tags, created_at, read_at FROM links WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get link: %w", err)
	}
	return row.toLink(), nil
}

// List retrieves links with optional filters.
func (s *SQLiteStorage) List(ctx context.Context, opts ListOptions) ([]*model.Link, error) {
	query := "SELECT id, url, title, note, tags, created_at, read_at FROM links WHERE 1=1"
	args := []interface{}{}

	switch opts.ReadStatus {
	case ReadStatusUnread:
		query += " AND read_at IS NULL"
	case ReadStatusRead:
		query += " AND read_at IS NOT NULL"
	}

	if opts.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%"+opts.Tag+"%")
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	var rows []linkRow
	err := s.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}

	links := make([]*model.Link, len(rows))
	for i := range rows {
		links[i] = rows[i].toLink()
	}

	return links, nil
}

// Delete removes a link by ID.
func (s *SQLiteStorage) Delete(ctx context.Context, id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	result, err := s.db.ExecContext(ctx, "DELETE FROM links WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	return checkRowsAffected(result, "delete link")
}

// MarkRead sets the read_at timestamp for a link.
func (s *SQLiteStorage) MarkRead(ctx context.Context, id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	result, err := s.db.ExecContext(ctx,
		"UPDATE links SET read_at = datetime('now') WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return checkRowsAffected(result, "mark read")
}

// MarkUnread clears the read_at timestamp for a link.
func (s *SQLiteStorage) MarkUnread(ctx context.Context, id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	result, err := s.db.ExecContext(ctx,
		"UPDATE links SET read_at = NULL WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("mark unread: %w", err)
	}
	return checkRowsAffected(result, "mark unread")
}

func checkRowsAffected(result sql.Result, action string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// Export returns all links for export.
func (s *SQLiteStorage) Export(ctx context.Context) ([]*model.Link, error) {
	return s.List(ctx, ListOptions{ReadStatus: ReadStatusAll})
}

// Import imports links from a slice, handling duplicates.
func (s *SQLiteStorage) Import(ctx context.Context, links []*model.Link) error {
	for _, link := range links {
		var existing linkRow
		err := s.db.GetContext(ctx, &existing,
			"SELECT id, url, title, note, tags, created_at, read_at FROM links WHERE url = ?", link.URL)

		var readAt sql.NullString
		if link.ReadAt != nil {
			readAt.String = link.ReadAt.Format(time.RFC3339)
			readAt.Valid = true
		}

		createdAtStr := link.CreatedAt.Format(time.RFC3339)
		if link.CreatedAt.IsZero() {
			createdAtStr = time.Now().Format(time.RFC3339)
		}

		if err == sql.ErrNoRows {
			// Generate ID if not provided
			if link.ID == "" {
				link.ID = model.GenerateShortID()
			}
			_, err = s.db.ExecContext(ctx,
				"INSERT INTO links (id, url, title, note, tags, created_at, read_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				link.ID, link.URL, link.Title, link.Note, link.Tags, createdAtStr, readAt)
			if err != nil {
				return fmt.Errorf("insert link %s: %w", link.URL, err)
			}
		} else if err != nil {
			return fmt.Errorf("check existing link %s: %w", link.URL, err)
		} else {
			existingLink := existing.toLink()
			// Preserve existing title/note if present, otherwise use new
			newTitle := existingLink.Title
			if newTitle == "" {
				newTitle = link.Title
			}
			newNote := existingLink.Note
			if newNote == "" {
				newNote = link.Note
			}

			if link.Tags != "" {
				mergeLink := &model.Link{Tags: link.Tags}
				existingLink.MergeTags(mergeLink)
			}
			newTags := existingLink.Tags

			_, err = s.db.ExecContext(ctx, "DELETE FROM links WHERE url = ?", link.URL)
			if err != nil {
				return fmt.Errorf("delete existing link %s: %w", link.URL, err)
			}

			existingCreatedAt := existingLink.CreatedAt.Format(time.RFC3339)
			if existingLink.CreatedAt.IsZero() {
				existingCreatedAt = createdAtStr
			}

			_, err = s.db.ExecContext(ctx,
				"INSERT INTO links (id, url, title, note, tags, created_at, read_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				existingLink.ID, link.URL, newTitle, newNote, newTags, existingCreatedAt, readAt)
			if err != nil {
				return fmt.Errorf("re-insert merged link %s: %w", link.URL, err)
			}
		}
	}

	return nil
}

// Search performs a full-text search across links.
func (s *SQLiteStorage) Search(ctx context.Context, query string) ([]*model.Link, error) {
	var rows []linkRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT l.id, l.url, l.title, l.note, l.tags, l.created_at, l.read_at
		FROM links l
		INNER JOIN links_fts ON l.rowid = links_fts.rowid
		WHERE links_fts MATCH ?
		ORDER BY l.created_at DESC
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search links: %w", err)
	}

	links := make([]*model.Link, len(rows))
	for i := range rows {
		links[i] = rows[i].toLink()
	}

	return links, nil
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func parseSQLiteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Time{}
}
