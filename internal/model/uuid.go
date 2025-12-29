package model

import (
	"encoding/base32"
	"strings"

	"github.com/google/uuid"
)

// GenerateShortID generates a short, URL-safe ID using UUID v4 encoded in base32.
func GenerateShortID() string {
	id := uuid.New()
	// Encode UUID bytes in base32 (no padding) for shorter representation
	// 16 bytes -> 26 base32 characters (URL-safe, case-insensitive)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(id[:])
	// Convert to lowercase for consistency
	return strings.ToLower(encoded)
}

// ValidateShortID validates that an ID is a valid format.
// Accepts both base32 encoded UUIDs (26 chars) and hex IDs from migration (12 chars).
// Case-insensitive to support IDs created before lowercase conversion.
func ValidateShortID(id string) bool {
	if len(id) < 10 || len(id) > 30 {
		return false
	}
	// Check if it's alphanumeric (case-insensitive)
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
