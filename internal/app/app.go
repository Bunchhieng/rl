package app

import (
	"os"
	"path/filepath"

	"github.com/bunchhieng/rl/internal/storage"
)

// DefaultDBPath returns the default database path using the platform's config directory.
func DefaultDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "rl", "links.db"), nil
}

// NewStorage creates a new storage instance with the default database path.
func NewStorage(dbPath string) (storage.Storage, error) {
	if dbPath == "" {
		var err error
		dbPath, err = DefaultDBPath()
		if err != nil {
			return nil, err
		}
	}
	return storage.NewSQLiteStorage(dbPath)
}
