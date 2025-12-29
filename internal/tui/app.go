package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bunchhieng/rl/internal/model"
	"github.com/bunchhieng/rl/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	estLocation *time.Location
)

func init() {
	var err error
	estLocation, err = time.LoadLocation("America/New_York")
	if err != nil {
		estLocation = time.UTC
	}
}

type appModel struct {
	storage       storage.Storage
	links         []*model.Link
	filtered      []*model.Link
	selected      int
	readStatus    storage.ReadStatus
	searchQuery   string
	searchMode    bool
	confirmDelete bool
	deleteLinkID  string
	width         int
	height        int
	err           error
	statusMsg     string
	statusTimer   *time.Timer
}

type loadLinksMsg struct {
	links []*model.Link
	err   error
}

type statusMsg struct {
	message string
}

func initialModel(s storage.Storage) appModel {
	return appModel{
		storage:    s,
		links:      []*model.Link{},
		filtered:   []*model.Link{},
		selected:   0,
		readStatus: storage.ReadStatusUnread,
		searchMode: false,
		width:      80,
		height:     24,
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		loadLinks(m.storage, m.readStatus),
		tea.EnterAltScreen,
	)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle delete confirmation first
	if m.confirmDelete {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return m.handleDeleteConfirmation(keyMsg)
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchInput(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "j", "down":
			m.moveDown()
			return m, nil

		case "k", "up":
			m.moveUp()
			return m, nil

		case "g":
			if len(msg.Runes) > 0 && msg.Runes[0] == 'g' {
				m.selected = 0
				return m, nil
			}

		case "G":
			m.selected = len(m.filtered) - 1
			if m.selected < 0 {
				m.selected = 0
			}
			return m, nil

		case "o", "enter":
			return m, m.openLink()

		case "d":
			return m, m.markRead()

		case "u":
			return m, m.markUnread()

		case "r":
			return m, m.promptDelete()

		case "/":
			m.searchMode = true
			m.searchQuery = ""
			return m, nil

		case "esc":
			m.searchMode = false
			m.searchQuery = ""
			m.applyFilters()
			return m, nil

		case "tab":
			m.cycleFilter()
			return m, loadLinks(m.storage, m.readStatus)

		case "a":
			return m, m.showAddLink()

		case "?":
			return m, m.showHelp()

		case "ctrl+l":
			return m, loadLinks(m.storage, m.readStatus)
		}

	case loadLinksMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.links = msg.links
		m.applyFilters()
		return m, nil

	case statusMsg:
		m.statusMsg = msg.message
		if m.statusTimer != nil {
			m.statusTimer.Stop()
		}
		m.statusTimer = time.NewTimer(3 * time.Second)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return statusMsg{""}
		})
	}

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Search bar
	if m.searchMode {
		searchBar := m.renderSearchBar()
		b.WriteString(searchBar)
		b.WriteString("\n")
	}

	// Links list
	list := m.renderList()
	b.WriteString(list)
	b.WriteString("\n")

	// Status bar
	statusBar := m.renderStatusBar()
	b.WriteString(statusBar)

	return b.String()
}

func (m *appModel) moveDown() {
	if m.selected < len(m.filtered)-1 {
		m.selected++
	}
}

func (m *appModel) moveUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *appModel) cycleFilter() {
	switch m.readStatus {
	case storage.ReadStatusUnread:
		m.readStatus = storage.ReadStatusRead
	case storage.ReadStatusRead:
		m.readStatus = storage.ReadStatusAll
	case storage.ReadStatusAll:
		m.readStatus = storage.ReadStatusUnread
	}
	m.selected = 0
}

func (m *appModel) applyFilters() {
	m.filtered = m.links

	// Apply search filter
	if m.searchQuery != "" {
		query := strings.ToLower(m.searchQuery)
		filtered := []*model.Link{}
		for _, link := range m.filtered {
			if strings.Contains(strings.ToLower(link.URL), query) ||
				strings.Contains(strings.ToLower(link.Title), query) ||
				strings.Contains(strings.ToLower(link.Note), query) ||
				strings.Contains(strings.ToLower(link.Tags), query) {
				filtered = append(filtered, link)
			}
		}
		m.filtered = filtered
	}

	// Ensure selected index is valid
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m *appModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		m.applyFilters()
		return m, nil

	case "enter":
		m.searchMode = false
		m.applyFilters()
		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.applyFilters()
		}
		return m, nil

	default:
		if len(msg.Runes) > 0 {
			m.searchQuery += string(msg.Runes)
			m.applyFilters()
		}
		return m, nil
	}
}

func loadLinks(s storage.Storage, readStatus storage.ReadStatus) tea.Cmd {
	return func() tea.Msg {
		links, err := s.List(context.Background(), storage.ListOptions{
			ReadStatus: readStatus,
		})
		return loadLinksMsg{links: links, err: err}
	}
}

func (m *appModel) openLink() tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return nil
	}

	link := m.filtered[m.selected]
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link.URL)
	case "linux":
		cmd = exec.Command("xdg-open", link.URL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", link.URL)
	default:
		return func() tea.Msg {
			return statusMsg{"Unsupported OS"}
		}
	}

	go cmd.Run()

	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Opened: %s", link.URL)}
	}
}

func (m *appModel) markRead() tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return nil
	}

	link := m.filtered[m.selected]
	if link.IsRead() {
		return func() tea.Msg {
			return statusMsg{"Already marked as read"}
		}
	}

	return tea.Batch(
		func() tea.Msg {
			err := m.storage.MarkRead(context.Background(), link.ID)
			if err != nil {
				return statusMsg{fmt.Sprintf("Error: %v", err)}
			}
			return statusMsg{"Marked as read"}
		},
		loadLinks(m.storage, m.readStatus),
	)
}

func (m *appModel) markUnread() tea.Cmd {
	// Try to get link from filtered list first
	var linkID string
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		link := m.filtered[m.selected]
		if !link.IsRead() {
			return func() tea.Msg {
				return statusMsg{"Already unread"}
			}
		}
		linkID = link.ID
	} else {
		// If not in filtered list, try to find the most recently read link in all links
		// This handles the case where a link was marked as read and disappeared from unread list
		var mostRecentRead *model.Link
		for _, link := range m.links {
			if link.IsRead() && link.ReadAt != nil {
				if mostRecentRead == nil || link.ReadAt.After(*mostRecentRead.ReadAt) {
					mostRecentRead = link
				}
			}
		}
		if mostRecentRead == nil {
			return func() tea.Msg {
				return statusMsg{"No read link found to mark as unread"}
			}
		}
		linkID = mostRecentRead.ID
	}

	return tea.Batch(
		func() tea.Msg {
			err := m.storage.MarkUnread(context.Background(), linkID)
			if err != nil {
				return statusMsg{fmt.Sprintf("Error: %v", err)}
			}
			return statusMsg{"Marked as unread"}
		},
		loadLinks(m.storage, m.readStatus),
	)
}

func (m *appModel) promptDelete() tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return nil
	}

	link := m.filtered[m.selected]
	m.confirmDelete = true
	m.deleteLinkID = link.ID
	return nil
}

func (m *appModel) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmDelete = false
		linkID := m.deleteLinkID
		m.deleteLinkID = ""
		return m, tea.Batch(
			func() tea.Msg {
				err := m.storage.Delete(context.Background(), linkID)
				if err != nil {
					return statusMsg{fmt.Sprintf("Error: %v", err)}
				}
				return statusMsg{"Deleted link"}
			},
			loadLinks(m.storage, m.readStatus),
		)

	case "n", "N", "esc":
		m.confirmDelete = false
		m.deleteLinkID = ""
		return m, nil

	default:
		return m, nil
	}
}

func (m *appModel) showAddLink() tea.Cmd {
	// TODO: Implement add link modal
	return func() tea.Msg {
		return statusMsg{"Add link: Not implemented yet. Use 'rl add <url>' from CLI"}
	}
}

func (m *appModel) showHelp() tea.Cmd {
	// TODO: Implement help screen
	return func() tea.Msg {
		return statusMsg{"Help: q=quit, j/k=nav, o=open, d=done, u=undo, r=remove, /=search, tab=filter"}
	}
}

// Run starts the TUI application
func Run(s storage.Storage) error {
	p := tea.NewProgram(initialModel(s), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
