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

// Commands handles all CLI command execution.
type Commands struct {
	storage storage.Storage
}

// NewCommands creates a new Commands instance.
func NewCommands(s storage.Storage) *Commands {
	return &Commands{storage: s}
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
		fmt.Printf("Updated link %s: %s\n", created.ID, created.URL)
	} else {
		fmt.Printf("Added link %s: %s\n", created.ID, created.URL)
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
		return handleNotFound(err, id, "get link")
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

	fmt.Printf("Opened: %s\n", link.URL)
	return nil
}

// Done marks a link as read.
func (c *Commands) Done(id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	if err := c.storage.MarkRead(context.Background(), id); err != nil {
		return handleNotFound(err, id, "mark read")
	}
	fmt.Printf("Marked link %s as read.\n", id)
	return nil
}

// Undo marks a link as unread.
func (c *Commands) Undo(id string) error {
	if !model.ValidateShortID(id) {
		return fmt.Errorf("invalid ID format")
	}
	if err := c.storage.MarkUnread(context.Background(), id); err != nil {
		return handleNotFound(err, id, "mark unread")
	}
	fmt.Printf("Marked link %s as unread.\n", id)
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
				failed = append(failed, fmt.Sprintf("%s (not found)", id))
			} else {
				failed = append(failed, fmt.Sprintf("%s (%v)", id, err))
			}
			continue
		}
		deleted = append(deleted, id)
	}

	if len(deleted) > 0 {
		if len(deleted) == 1 {
			fmt.Printf("Deleted link %s.\n", deleted[0])
		} else {
			fmt.Printf("Deleted %d link(s): %s\n", len(deleted), strings.Join(deleted, ", "))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to delete: %s", strings.Join(failed, ", "))
	}

	return nil
}

func handleNotFound(err error, id string, action string) error {
	if err == model.ErrNotFound {
		return fmt.Errorf("link %s not found", id)
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

	fmt.Printf("Imported %d link(s).\n", len(links))
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

	header := fmt.Sprintf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │",
		colIDLen-2, "ID",
		colURLLen-2, "URL",
		colTitleLen-2, "TITLE",
		colCreatedLen-2, "CREATED",
		colTagsLen-2, "TAGS")

	separator := fmt.Sprintf("├%s┼%s┼%s┼%s┼%s┤",
		strings.Repeat("─", colIDLen),
		strings.Repeat("─", colURLLen),
		strings.Repeat("─", colTitleLen),
		strings.Repeat("─", colCreatedLen),
		strings.Repeat("─", colTagsLen))

	topBorder := fmt.Sprintf("┌%s┐", strings.Repeat("─", totalWidth))
	bottomBorder := fmt.Sprintf("└%s┘", strings.Repeat("─", totalWidth))

	fmt.Println(topBorder)
	fmt.Println(header)
	fmt.Println(separator)

	for _, link := range links {
		url := truncateString(link.URL, colURLLen-2)
		title := truncateString(link.Title, colTitleLen-2)
		tags := truncateString(link.Tags, colTagsLen-2)
		created := formatTime(link.CreatedAt)

		row := fmt.Sprintf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │",
			colIDLen-2, link.ID,
			colURLLen-2, url,
			colTitleLen-2, title,
			colCreatedLen-2, created,
			colTagsLen-2, tags)
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

// Version prints the version.
func (c *Commands) Version(version string) {
	fmt.Printf("rl version %s\n", version)
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
