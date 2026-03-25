package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"

	"github.com/adruban/osem/internal/bookmarks"
	"github.com/adruban/osem/internal/db"
	"github.com/adruban/osem/internal/models"
	"github.com/adruban/osem/internal/tmux"
)

type sessionItem struct {
	session      models.Session
	isTmuxActive bool
}

func (i sessionItem) FilterValue() string { return i.session.Title }
func (i sessionItem) Title() string {
	prefix := ""
	if i.isTmuxActive {
		prefix = "● "
	}
	return prefix + i.session.Title
}
func (i sessionItem) Description() string {
	return fmt.Sprintf("%s • %s",
		truncateDir(i.session.Directory, 40),
		i.session.UpdatedAt.Format("2006-01-02 15:04"),
	)
}

func truncateDir(dir string, maxLen int) string {
	if len(dir) <= maxLen {
		return dir
	}
	home := "~"
	if strings.HasPrefix(dir, "/home/") {
		parts := strings.SplitN(dir, "/", 4)
		if len(parts) >= 4 {
			home = "~/" + parts[3]
		}
		if len(home) <= maxLen {
			return home
		}
	}
	if len(dir) > maxLen {
		return "..." + dir[len(dir)-maxLen+3:]
	}
	return dir
}

type keymap struct {
	selectKey key.Binding
	delete    key.Binding
	bookmark  key.Binding
	rename    key.Binding
	search    key.Binding
	quit      key.Binding
}

func defaultKeymap() keymap {
	return keymap{
		selectKey: key.NewBinding(
			key.WithKeys("enter", "s"),
			key.WithHelp("enter/s", "select"),
		),
		delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		bookmark: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "bookmark"),
		),
		rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
	}
}

type Model struct {
	list      list.Model
	db        *db.Client
	bookmarks *bookmarks.Manager
	tmuxMgr   *tmux.Manager
	sessions  []models.Session
	selected  *models.Session
	quitting  bool
	err       error
	keys      keymap
	width     int
	height    int
}

type sessionLoadedMsg []models.Session
type errorMsg error

func New(dbClient *db.Client, bmMgr *bookmarks.Manager) (*Model, error) {
	tmuxMgr := tmux.NewManager("opencode-")
	keys := defaultKeymap()

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = descStyle
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "OpenCode Sessions"
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	l.SetFilteringEnabled(true)

	m := &Model{
		list:      l,
		db:        dbClient,
		bookmarks: bmMgr,
		tmuxMgr:   tmuxMgr,
		keys:      keys,
	}

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return m.loadSessions
}

func (m *Model) loadSessions() tea.Msg {
	sessions, err := m.db.ListSessions(100)
	if err != nil {
		return errorMsg(err)
	}
	return sessionLoadedMsg(sessions)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil

	case sessionLoadedMsg:
		m.sessions = msg
		items := make([]list.Item, len(msg))
		activeSessions, _ := m.tmuxMgr.ListOpencodeSessions()
		for i, s := range msg {
			shortID := s.ShortID()
			isActive := false
			for _, as := range activeSessions {
				if strings.Contains(as, shortID) {
					isActive = true
					break
				}
			}
			items[i] = sessionItem{session: s, isTmuxActive: isActive}
		}
		return m, m.list.SetItems(items)

	case errorMsg:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.selectKey):
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				m.selected = &item.session
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.delete):
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				if err := m.db.DeleteSession(item.session.ID); err != nil {
					m.err = err
					return m, nil
				}
				return m, m.loadSessions
			}

		case key.Matches(msg, m.keys.bookmark):
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				if m.bookmarks.IsBookmarked(item.session.ID) {
					m.bookmarks.Remove(item.session.ID)
				} else {
					m.bookmarks.Add(item.session.ID, "")
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Main list
	b.WriteString(m.list.View())

	// Footer with help
	b.WriteString("\n")
	b.WriteString(renderHelp(m.keys))

	return b.String()
}

func renderHelp(keys keymap) string {
	help := strings.Join([]string{
		"enter: select",
		"d: delete",
		"f: bookmark",
		"/: search",
		"q: quit",
	}, " • ")
	return helpStyle.Render(help)
}

func (m *Model) SelectedSession() *models.Session {
	return m.selected
}
