package model

import "errors"

var (
	// ErrNotFound indicates a link was not found.
	ErrNotFound = errors.New("link not found")

	// ErrInvalidURL indicates an invalid URL was provided.
	ErrInvalidURL = errors.New("invalid URL")

	// ErrDuplicate indicates a duplicate URL already exists.
	ErrDuplicate = errors.New("duplicate URL")
)
