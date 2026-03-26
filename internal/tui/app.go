package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adruban/osem/internal/bookmarks"
	"github.com/adruban/osem/internal/config"
	"github.com/adruban/osem/internal/db"
	"github.com/adruban/osem/internal/models"
	"github.com/adruban/osem/internal/tmux"
	"github.com/adruban/osem/internal/tui/theme"
)

type sessionItem struct {
	session      models.Session
	isTmuxActive bool
	isBookmark   bool
	groupName    string
}

func (i sessionItem) FilterValue() string {
	return i.session.Title
}

func (i sessionItem) Title() string {
	prefix := ""
	if i.isBookmark {
		prefix = "★ "
	} else if i.isTmuxActive {
		prefix = "● "
	}
	return prefix + i.session.Title
}

func (i sessionItem) Description() string {
	return fmt.Sprintf("%s • %s",
		truncDir(i.session.Directory, 40),
		i.session.UpdatedAt.Format("2006-01-02 15:04"),
	)
}

type keymap struct {
	selectKey     key.Binding
	delete        key.Binding
	bookmark      key.Binding
	toggleBm      key.Binding
	togglePreview key.Binding
	nextTheme     key.Binding
	search        key.Binding
	quit          key.Binding
	nextGroup     key.Binding
	prevGroup     key.Binding
	group1        key.Binding
	group2        key.Binding
	group3        key.Binding
	group4        key.Binding
	group5        key.Binding
	group6        key.Binding
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
		toggleBm: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "bookmarks"),
		),
		togglePreview: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "preview"),
		),
		nextTheme: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "theme"),
		),
		search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
		nextGroup: key.NewBinding(
			key.WithKeys("n", "right"),
			key.WithHelp("n/→", "next"),
		),
		prevGroup: key.NewBinding(
			key.WithKeys("p", "left"),
			key.WithHelp("p/←", "prev"),
		),
		group1: key.NewBinding(key.WithKeys("1")),
		group2: key.NewBinding(key.WithKeys("2")),
		group3: key.NewBinding(key.WithKeys("3")),
		group4: key.NewBinding(key.WithKeys("4")),
		group5: key.NewBinding(key.WithKeys("5")),
		group6: key.NewBinding(key.WithKeys("6")),
	}
}

type sessionGroup struct {
	Name     string
	Sessions []models.Session
}

type Model struct {
	list         list.Model
	db           *db.Client
	bookmarks    *bookmarks.Manager
	tmuxMgr      *tmux.Manager
	sessions     []models.Session
	groups       []sessionGroup
	currentGroup int
	bookmarkMode bool
	selected     *models.Session
	quitting     bool
	err          error
	keys         keymap
	width        int
	height       int
	theme        theme.Theme
	themeName    string
	config       *config.Config
	preview      *Preview
	showPreview  bool
}

type sessionLoadedMsg []models.Session
type errorMsg error

func New(dbClient *db.Client, bmMgr *bookmarks.Manager) (*Model, error) {
	tmuxMgr := tmux.NewManager("opencode-")
	keys := defaultKeymap()

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.DefaultConfig
	}

	themeName := cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	t := theme.Get(themeName)

	preview := NewPreview(t)

	delegate := newItemDelegate(t)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "OpenCode Sessions"
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)
	l.Styles.HelpStyle = lipgloss.NewStyle().
		Foreground(t.TextMuted)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	m := &Model{
		list:        l,
		db:          dbClient,
		bookmarks:   bmMgr,
		tmuxMgr:     tmuxMgr,
		keys:        keys,
		theme:       t,
		themeName:   themeName,
		config:      cfg,
		preview:     preview,
		showPreview: cfg.ShowPreview,
	}

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return m.loadSessions
}

func (m *Model) loadSessions() tea.Msg {
	sessions, err := m.db.ListSessions(500)
	if err != nil {
		return errorMsg(err)
	}
	return sessionLoadedMsg(sessions)
}

func (m *Model) groupSessions(sessions []models.Session) []sessionGroup {
	var bookmarks, today, yesterday, thisWeek, thisMonth, older []models.Session
	now := time.Now()

	for _, s := range sessions {
		if m.bookmarks != nil && m.bookmarks.IsBookmarked(s.ID) {
			bookmarks = append(bookmarks, s)
			continue
		}

		age := now.Sub(s.UpdatedAt)
		switch {
		case age < 24*time.Hour:
			today = append(today, s)
		case age < 48*time.Hour:
			yesterday = append(yesterday, s)
		case age < 7*24*time.Hour:
			thisWeek = append(thisWeek, s)
		case age < 30*24*time.Hour:
			thisMonth = append(thisMonth, s)
		default:
			older = append(older, s)
		}
	}

	var groups []sessionGroup
	if len(bookmarks) > 0 {
		groups = append(groups, sessionGroup{Name: "★ Bookmarks", Sessions: bookmarks})
	}
	if len(today) > 0 {
		groups = append(groups, sessionGroup{Name: "Today", Sessions: today})
	}
	if len(yesterday) > 0 {
		groups = append(groups, sessionGroup{Name: "Yesterday", Sessions: yesterday})
	}
	if len(thisWeek) > 0 {
		groups = append(groups, sessionGroup{Name: "This Week", Sessions: thisWeek})
	}
	if len(thisMonth) > 0 {
		groups = append(groups, sessionGroup{Name: "This Month", Sessions: thisMonth})
	}
	if len(older) > 0 {
		groups = append(groups, sessionGroup{Name: "Older", Sessions: older})
	}

	return groups
}

func (m *Model) cycleTheme() {
	themes := theme.Names()
	currentIdx := 0
	for i, name := range themes {
		if name == m.themeName {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + 1) % len(themes)
	m.themeName = themes[nextIdx]
	m.theme = theme.Get(m.themeName)

	m.config.Theme = m.themeName
	config.Save(m.config)

	m.preview.Theme = m.theme
	delegate := newItemDelegate(m.theme)
	m.list.SetDelegate(delegate)
	m.list.Styles.Title = lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Padding(0, 1)
	m.list.Styles.HelpStyle = lipgloss.NewStyle().
		Foreground(m.theme.TextMuted)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case sessionLoadedMsg:
		m.sessions = msg
		m.groups = m.groupSessions(msg)
		if len(m.groups) > 0 {
			m.currentGroup = 0
			m.updateList()
		}
		return m, nil

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
				m.groups = m.groupSessions(m.sessions)
				m.updateList()
				return m, nil
			}

		case key.Matches(msg, m.keys.toggleBm):
			m.bookmarkMode = !m.bookmarkMode
			m.currentGroup = 0
			m.groups = m.groupSessions(m.sessions)
			m.updateList()
			return m, nil

		case key.Matches(msg, m.keys.togglePreview):
			m.showPreview = !m.showPreview
			m.config.ShowPreview = m.showPreview
			config.Save(m.config)
			m.updateSizes()
			return m, nil

		case key.Matches(msg, m.keys.nextTheme):
			m.cycleTheme()
			return m, nil

		case key.Matches(msg, m.keys.nextGroup):
			if len(m.groups) > 0 && m.currentGroup < len(m.groups)-1 {
				m.currentGroup++
				m.updateList()
			}
			return m, nil

		case key.Matches(msg, m.keys.prevGroup):
			if m.currentGroup > 0 {
				m.currentGroup--
				m.updateList()
			}
			return m, nil

		case key.Matches(msg, m.keys.group1):
			m.jumpToGroup(0)
			return m, nil
		case key.Matches(msg, m.keys.group2):
			m.jumpToGroup(1)
			return m, nil
		case key.Matches(msg, m.keys.group3):
			m.jumpToGroup(2)
			return m, nil
		case key.Matches(msg, m.keys.group4):
			m.jumpToGroup(3)
			return m, nil
		case key.Matches(msg, m.keys.group5):
			m.jumpToGroup(4)
			return m, nil
		case key.Matches(msg, m.keys.group6):
			m.jumpToGroup(5)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.updatePreview()
	return m, cmd
}

func (m *Model) jumpToGroup(idx int) {
	if idx >= 0 && idx < len(m.groups) {
		m.currentGroup = idx
		m.updateList()
	}
}

func (m *Model) updateSizes() {
	if m.showPreview {
		listWidth := m.width * 60 / 100
		previewWidth := m.width - listWidth - 3
		m.list.SetSize(listWidth, m.height-5)
		m.preview.SetSize(previewWidth, m.height-5)
	} else {
		m.list.SetSize(m.width, m.height-5)
	}
}

func (m *Model) updateList() {
	if len(m.groups) == 0 || m.currentGroup >= len(m.groups) {
		return
	}

	group := m.groups[m.currentGroup]
	items := make([]list.Item, 0, len(group.Sessions))

	activeSessions, _ := m.tmuxMgr.ListOpencodeSessions()

	for _, s := range group.Sessions {
		shortID := s.ShortID()
		isActive := false
		for _, as := range activeSessions {
			if strings.Contains(as, shortID) {
				isActive = true
				break
			}
		}
		items = append(items, sessionItem{
			session:      s,
			isTmuxActive: isActive,
			isBookmark:   m.bookmarks != nil && m.bookmarks.IsBookmarked(s.ID),
			groupName:    group.Name,
		})
	}

	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s (%d)", group.Name, len(group.Sessions))
}

func (m *Model) updatePreview() {
	if !m.showPreview {
		return
	}
	if item, ok := m.list.SelectedItem().(sessionItem); ok {
		msgCount, _ := m.db.GetMessageCount(item.session.ID)
		m.preview.SetSession(&item.session, item.isBookmark, item.isTmuxActive, msgCount)
	} else {
		m.preview.SetSession(nil, false, false, 0)
	}
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Foreground(m.theme.Error).
			Padding(0, 1)
		b.WriteString(errStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Header with group navigation
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Main content (list + optional preview)
	var content string
	if m.showPreview {
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			m.list.View(),
			lipgloss.NewStyle().
				Foreground(m.theme.Border).
				Render("│"),
			m.preview.View(),
		)
	} else {
		content = m.list.View()
	}
	b.WriteString(content)

	// Footer with help
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) renderHeader() string {
	themeIndicator := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Render(fmt.Sprintf("[%s]", m.theme.Name))

	// Group navigation
	if len(m.groups) > 0 {
		nav := make([]string, len(m.groups))
		for i, g := range m.groups {
			style := lipgloss.NewStyle()
			if i == m.currentGroup {
				style = style.Foreground(m.theme.Primary).Bold(true)
				nav[i] = style.Render(fmt.Sprintf("[%d] %s (%d)", i+1, g.Name, len(g.Sessions)))
			} else {
				style = style.Foreground(m.theme.TextMuted)
				nav[i] = style.Render(fmt.Sprintf("(%d) %s", i+1, g.Name))
			}
		}
		groupNav := strings.Join(nav, " ")
		return lipgloss.NewStyle().Width(m.width).Render(
			groupNav + " " + themeIndicator,
		)
	}

	return themeIndicator
}

func (m *Model) renderFooter() string {
	var parts []string

	parts = append(parts, "enter:select")
	parts = append(parts, "d:delete")
	parts = append(parts, "f:bookmark")

	if m.bookmarkMode {
		parts = append(parts, "b:all")
	} else {
		parts = append(parts, "b:bookmarks")
	}

	if m.showPreview {
		parts = append(parts, "tab:hide")
	} else {
		parts = append(parts, "tab:preview")
	}

	parts = append(parts, "t:theme")
	parts = append(parts, "1-6:groups")
	parts = append(parts, "/:search")
	parts = append(parts, "q:quit")

	return lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Render(strings.Join(parts, " • "))
}

func (m *Model) SelectedSession() *models.Session {
	return m.selected
}
