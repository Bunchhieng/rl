package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/bunchhieng/rl/internal/model"
	"github.com/bunchhieng/rl/internal/storage"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	unreadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	readStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	searchStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
)

func (m appModel) renderHeader() string {
	filterText := "Unread"
	switch m.readStatus {
	case storage.ReadStatusRead:
		filterText = "Read"
	case storage.ReadStatusAll:
		filterText = "All"
	}

	header := fmt.Sprintf("rl - Read Later  [Filter: %s]  [%d links]", filterText, len(m.filtered))
	return headerStyle.Render(header)
}

func (m appModel) renderSearchBar() string {
	prompt := fmt.Sprintf("/%s", m.searchQuery)
	return searchStyle.Width(m.width - 2).Render(prompt)
}

func (m appModel) renderList() string {
	if m.confirmDelete {
		return m.renderDeleteConfirmation()
	}

	if len(m.filtered) == 0 {
		return "No links found. Press 'a' to add a link or 'q' to quit."
	}

	var b strings.Builder
	listHeight := m.height - 6 // Reserve space for header, search, status

	for i, link := range m.filtered {
		if i >= listHeight {
			break
		}

		line := m.renderLink(link, i == m.selected)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m appModel) renderLink(link *model.Link, selected bool) string {
	// Status indicator
	statusIcon := "○"
	statusColor := unreadStyle
	if link.IsRead() {
		statusIcon = "●"
		statusColor = readStyle
	}

	// Title or URL
	title := link.Title
	if title == "" {
		title = link.URL
	}
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	// Format time
	timeStr := formatTime(link.CreatedAt)

	// Tags
	tagsStr := ""
	if link.Tags != "" {
		tagsStr = fmt.Sprintf(" [%s]", link.Tags)
	}

	// Build line
	line := fmt.Sprintf("%s %s %s%s",
		statusColor.Render(statusIcon),
		urlStyle.Render(title),
		readStyle.Render(timeStr),
		tagStyle.Render(tagsStr),
	)

	if selected {
		line = selectedStyle.Render(line)
	} else {
		// Add padding to match selected style width
		line = " " + line
	}

	return line
}

func (m appModel) renderStatusBar() string {
	var parts []string

	if m.statusMsg != "" {
		parts = append(parts, m.statusMsg)
	} else {
		parts = append(parts, fmt.Sprintf("%d/%d", m.selected+1, len(m.filtered)))
	}

	parts = append(parts, "[o]pen [d]one [u]ndo [r]emove [tab]filter [q]uit")

	return statusBarStyle.Width(m.width).Render(strings.Join(parts, "  |  "))
}

func (m appModel) renderDeleteConfirmation() string {
	var linkTitle string
	if m.selected < len(m.filtered) {
		link := m.filtered[m.selected]
		linkTitle = link.Title
		if linkTitle == "" {
			linkTitle = link.URL
		}
		if len(linkTitle) > 50 {
			linkTitle = linkTitle[:47] + "..."
		}
	}

	confirmText := fmt.Sprintf("Delete link: %s?\n\n[y]es / [n]o", linkTitle)
	return selectedStyle.Width(m.width-4).Padding(1, 2).Render(confirmText)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(estLocation).Format("2006-01-02 15:04")
}
