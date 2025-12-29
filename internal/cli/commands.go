package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bunchhieng/rl/internal/model"
	"github.com/bunchhieng/rl/internal/storage"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Commands handles all CLI command execution.
type Commands struct {
	storage storage.Storage
}

// NewCommands creates a new Commands instance.
func NewCommands(s storage.Storage) *Commands {
	return &Commands{storage: s}
}

// suggestID suggests a similar ID if the given ID is not found.
func (c *Commands) suggestID(id string) string {
	// Get all links to find similar IDs
	links, err := c.storage.List(context.Background(), storage.ListOptions{
		ReadStatus: storage.ReadStatusAll,
	})
	if err != nil {
		return ""
	}

	if len(links) == 0 {
		return ""
	}

	bestMatch := ""
	minDistance := len(id) + 1

	for _, link := range links {
		distance := levenshteinDistance(id, link.ID)
		if distance < minDistance && distance <= 3 {
			minDistance = distance
			bestMatch = link.ID
		}
	}

	return bestMatch
}

func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// Add adds a new link.
func (c *Commands) Add(url string, title, note, tags string) error {
	link := &model.Link{
		URL:   url,
		Title: title,
		Note:  note,
		Tags:  tags,
	}

	if err := link.Validate(); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check if link already exists by trying to find it
	allLinks, _ := c.storage.List(context.Background(), storage.ListOptions{
		ReadStatus: storage.ReadStatusAll,
	})

	wasUpdate := false
	for _, existing := range allLinks {
		if existing.URL == url {
			wasUpdate = true
			break
		}
	}

	created, err := c.storage.Add(context.Background(), link)
	if err != nil {
		return fmt.Errorf("add link: %w", err)
	}

	if wasUpdate {
		fmt.Printf("%sUpdated%s link %s%s%s: %s%s%s\n", colorYellow, colorReset, colorBold, created.ID, colorReset, colorCyan, created.URL, colorReset)
	} else {
		fmt.Printf("%sAdded%s link %s%s%s: %s%s%s\n", colorGreen, colorReset, colorBold, created.ID, colorReset, colorCyan, created.URL, colorReset)
	}
	return nil
}

// List lists links with optional filters.
func (c *Commands) List(readStatus storage.ReadStatus, tag string, limit int) error {
	opts := storage.ListOptions{
		ReadStatus: readStatus,
		Tag:        tag,
		Limit:      limit,
	}

	links, err := c.storage.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("list links: %w", err)
	}

	if len(links) == 0 {
		fmt.Println("No links found.")
		return nil
	}

	return printLinksTable(links)
}

// Open opens a link in the default browser.
func (c *Commands) Open(id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	link, err := c.storage.Get(context.Background(), id)
	if err != nil {
		return c.handleNotFound(err, id, "get link")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link.URL)
	case "linux":
		cmd = exec.Command("xdg-open", link.URL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", link.URL)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	fmt.Printf("%sOpened:%s %s%s%s\n", colorGreen, colorReset, colorCyan, link.URL, colorReset)
	return nil
}

// Done marks a link as read.
func (c *Commands) Done(id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	if err := c.storage.MarkRead(context.Background(), id); err != nil {
		return c.handleNotFound(err, id, "mark read")
	}
	fmt.Printf("%sMarked%s link %s%s%s as read.\n", colorGreen, colorReset, colorBold, id, colorReset)
	return nil
}

// Undo marks a link as unread.
func (c *Commands) Undo(id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	if err := c.storage.MarkUnread(context.Background(), id); err != nil {
		return c.handleNotFound(err, id, "mark unread")
	}
	fmt.Printf("%sMarked%s link %s%s%s as unread.\n", colorYellow, colorReset, colorBold, id, colorReset)
	return nil
}

// Remove deletes one or more links.
func (c *Commands) Remove(ids ...string) error {
	if len(ids) == 0 {
		return fmt.Errorf("at least one ID required")
	}

	var deleted []string
	var failed []string

	for _, id := range ids {
		if !model.ValidateShortID(id) {
			failed = append(failed, fmt.Sprintf("%s (invalid format)", id))
			continue
		}
		if err := c.storage.Delete(context.Background(), id); err != nil {
			if err == model.ErrNotFound {
				suggestion := c.suggestID(id)
				msg := fmt.Sprintf("%s (not found)", id)
				if suggestion != "" {
					msg += fmt.Sprintf(" - %sDid you mean:%s %s%s%s?", colorYellow, colorReset, colorBold, suggestion, colorReset)
				}
				failed = append(failed, msg)
			} else {
				failed = append(failed, fmt.Sprintf("%s (%v)", id, err))
			}
			continue
		}
		deleted = append(deleted, id)
	}

	if len(deleted) > 0 {
		if len(deleted) == 1 {
			fmt.Printf("%sDeleted%s link %s%s%s.\n", colorRed, colorReset, colorBold, deleted[0], colorReset)
		} else {
			ids := strings.Join(deleted, ", ")
			fmt.Printf("%sDeleted%s %d link(s): %s%s%s\n", colorRed, colorReset, len(deleted), colorBold, ids, colorReset)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to delete: %s", strings.Join(failed, ", "))
	}

	return nil
}

func (c *Commands) handleNotFound(err error, id string, action string) error {
	if err == model.ErrNotFound {
		// Try to suggest similar IDs
		suggestion := c.suggestID(id)
		msg := fmt.Sprintf("link %s%s%s not found", colorBold, id, colorReset)
		if suggestion != "" {
			msg += fmt.Sprintf("\n\n%sDid you mean:%s %s%s%s?", colorYellow, colorReset, colorBold, suggestion, colorReset)
		}
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s: %w", action, err)
}

// Export exports all links to JSON.
func (c *Commands) Export(w io.Writer) error {
	links, err := c.storage.Export(context.Background())
	if err != nil {
		return fmt.Errorf("export links: %w", err)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(links); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	return nil
}

// Import imports links from a JSON file.
func (c *Commands) Import(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var links []*model.Link
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&links); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	if err := c.storage.Import(context.Background(), links); err != nil {
		return fmt.Errorf("import links: %w", err)
	}

	fmt.Printf("%sImported%s %s%d%s link(s).\n", colorGreen, colorReset, colorBold, len(links), colorReset)
	return nil
}

// Search performs a full-text search.
func (c *Commands) Search(query string) error {
	links, err := c.storage.Search(context.Background(), query)
	if err != nil {
		return fmt.Errorf("search links: %w", err)
	}

	if len(links) == 0 {
		fmt.Println("No links found.")
		return nil
	}

	return printLinksTable(links)
}

const (
	maxURLLen   = 60
	maxTitleLen = 40
	maxTagsLen  = 30
	ellipsisLen = 3
)

var estLocation *time.Location

func init() {
	var err error
	estLocation, err = time.LoadLocation("America/New_York")
	if err != nil {
		estLocation = time.UTC
	}
}

func printLinksTable(links []*model.Link) error {
	// Calculate column widths based on header and content
	colIDLen := len("ID")
	colURLLen := len("URL")
	colTitleLen := len("TITLE")
	colCreatedLen := len("CREATED")
	colTagsLen := len("TAGS")

	// Find maximum content widths (with limits)
	for _, link := range links {
		if idLen := len(link.ID); idLen > colIDLen {
			colIDLen = idLen
		}
		urlLen := truncateLen(len(link.URL), maxURLLen)
		if urlLen > colURLLen {
			colURLLen = urlLen
		}
		titleLen := truncateLen(len(link.Title), maxTitleLen)
		if titleLen > colTitleLen {
			colTitleLen = titleLen
		}
		createdLen := len(formatTime(link.CreatedAt))
		if createdLen > colCreatedLen {
			colCreatedLen = createdLen
		}
		tagsLen := truncateLen(len(link.Tags), maxTagsLen)
		if tagsLen > colTagsLen {
			colTagsLen = tagsLen
		}
	}

	// Add padding (2 spaces: one before, one after)
	colIDLen += 2
	colURLLen += 2
	colTitleLen += 2
	colCreatedLen += 2
	colTagsLen += 2

	// Calculate total width: sum of columns + 4 separators (│) + 2 spaces per separator
	totalWidth := colIDLen + colURLLen + colTitleLen + colCreatedLen + colTagsLen + 4

	header := fmt.Sprintf("%s│%s %s%-*s%s │ %s%-*s%s │ %s%-*s%s │ %s%-*s%s │ %s%-*s%s %s│%s",
		colorDim, colorReset,
		colorBold, colIDLen-2, "ID", colorReset,
		colorBold, colURLLen-2, "URL", colorReset,
		colorBold, colTitleLen-2, "TITLE", colorReset,
		colorBold, colCreatedLen-2, "CREATED", colorReset,
		colorBold, colTagsLen-2, "TAGS", colorReset,
		colorDim, colorReset)

	separator := fmt.Sprintf("%s├%s┼%s┼%s┼%s┼%s┤%s",
		colorDim,
		strings.Repeat("─", colIDLen),
		strings.Repeat("─", colURLLen),
		strings.Repeat("─", colTitleLen),
		strings.Repeat("─", colCreatedLen),
		strings.Repeat("─", colTagsLen),
		colorReset)

	topBorder := fmt.Sprintf("%s┌%s┐%s", colorDim, strings.Repeat("─", totalWidth), colorReset)
	bottomBorder := fmt.Sprintf("%s└%s┘%s", colorDim, strings.Repeat("─", totalWidth), colorReset)

	fmt.Println(topBorder)
	fmt.Println(header)
	fmt.Println(separator)

	for _, link := range links {
		url := truncateString(link.URL, colURLLen-2)
		title := truncateString(link.Title, colTitleLen-2)
		tags := truncateString(link.Tags, colTagsLen-2)
		created := formatTime(link.CreatedAt)

		idColor := colorBold + colorCyan
		row := fmt.Sprintf("%s│%s %s%-*s%s │ %s%-*s%s │ %-*s │ %s%-*s%s │ %s%-*s%s %s│%s",
			colorDim, colorReset,
			idColor, colIDLen-2, link.ID, colorReset,
			colorCyan, colURLLen-2, url, colorReset,
			colTitleLen-2, title,
			colorDim, colCreatedLen-2, created, colorReset,
			colorYellow, colTagsLen-2, tags, colorReset,
			colorDim, colorReset)
		fmt.Println(row)
	}

	fmt.Println(bottomBorder)
	return nil
}

func truncateLen(n, max int) int {
	if n > max {
		return max
	}
	return n
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-ellipsisLen] + "..."
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(estLocation).Format("2006-01-02 15:04:05 EST")
}

// ParseID validates an ID string format.
func ParseID(s string) (string, error) {
	if !model.ValidateShortID(s) {
		return "", fmt.Errorf("invalid ID format: %s", s)
	}
	return s, nil
}
