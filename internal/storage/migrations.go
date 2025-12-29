package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	migrationFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	for _, filename := range migrationFiles {
		parts := strings.Split(filename, "_")
		if len(parts) == 0 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		if applied[version] {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))",
			version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}
	}

	return nil
}

func getAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	applied := make(map[int]bool)

	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create schema_migrations table: %w", err)
	}

	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, rows.Err()
}
