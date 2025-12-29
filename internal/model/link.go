package model

import (
	"net/url"
	"strings"
	"time"
)

// Link represents a saved URL with metadata.
type Link struct {
	ID        string     `json:"id"`
	URL       string     `json:"url"`
	Title     string     `json:"title,omitempty"`
	Note      string     `json:"note,omitempty"`
	Tags      string     `json:"tags,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

// Validate checks if the link has a valid URL.
func (l *Link) Validate() error {
	if l.URL == "" {
		return ErrInvalidURL
	}
	u, err := url.Parse(l.URL)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme == "" || u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

// IsRead returns true if the link has been marked as read.
func (l *Link) IsRead() bool {
	return l.ReadAt != nil
}

// TagList returns tags as a slice of strings.
func (l *Link) TagList() []string {
	if l.Tags == "" {
		return nil
	}
	tags := strings.Split(l.Tags, ",")
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

// MergeTags merges tags from another link, deduplicating.
func (l *Link) MergeTags(other *Link) {
	if other.Tags == "" {
		return
	}
	existing := make(map[string]bool)
	for _, tag := range l.TagList() {
		existing[strings.ToLower(tag)] = true
	}

	newTags := l.TagList()
	for _, tag := range other.TagList() {
		tagLower := strings.ToLower(tag)
		if !existing[tagLower] {
			existing[tagLower] = true
			newTags = append(newTags, tag)
		}
	}
	l.Tags = strings.Join(newTags, ",")
}
