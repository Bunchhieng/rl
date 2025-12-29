package storage

import (
	"context"

	"github.com/bunchhieng/rl/internal/model"
)

// Storage defines the interface for link storage operations.
type Storage interface {
	// Add creates a new link.
	Add(ctx context.Context, link *model.Link) (*model.Link, error)

	// Get retrieves a link by ID.
	Get(ctx context.Context, id string) (*model.Link, error)

	// List retrieves links with optional filters.
	List(ctx context.Context, opts ListOptions) ([]*model.Link, error)

	// Delete removes a link by ID.
	Delete(ctx context.Context, id string) error

	// MarkRead sets the read_at timestamp for a link.
	MarkRead(ctx context.Context, id string) error

	// MarkUnread clears the read_at timestamp for a link.
	MarkUnread(ctx context.Context, id string) error

	// Export returns all links for export.
	Export(ctx context.Context) ([]*model.Link, error)

	// Import imports links from a slice, handling duplicates.
	Import(ctx context.Context, links []*model.Link) error

	// Search performs a full-text search across links.
	Search(ctx context.Context, query string) ([]*model.Link, error)

	// Close closes the storage connection.
	Close() error
}

// ListOptions specifies filtering options for List.
type ListOptions struct {
	ReadStatus ReadStatus
	Tag        string
	Limit      int
}

// ReadStatus indicates which links to include.
type ReadStatus int

const (
	ReadStatusUnread ReadStatus = iota
	ReadStatusRead
	ReadStatusAll
)
