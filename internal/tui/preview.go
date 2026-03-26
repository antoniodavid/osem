package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/adruban/osem/internal/models"
	"github.com/adruban/osem/internal/tui/theme"
)

type Preview struct {
	Session      *models.Session
	TmuxActive   bool
	IsBookmark   bool
	MessageCount int
	Width        int
	Height       int
	Theme        theme.Theme
}

func NewPreview(t theme.Theme) *Preview {
	return &Preview{Theme: t}
}

func (p *Preview) SetSession(s *models.Session, isBookmark, tmuxActive bool, msgCount int) {
	p.Session = s
	p.IsBookmark = isBookmark
	p.TmuxActive = tmuxActive
	p.MessageCount = msgCount
}

func (p *Preview) SetSize(width, height int) {
	p.Width = width
	p.Height = height
}

func (p *Preview) View() string {
	if p.Session == nil {
		return p.emptyView()
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Theme.Primary).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Session Details"))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Foreground(p.Theme.TextMuted).
		Width(10).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(p.Theme.Text)

	pair := func(label, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render(label+":"),
			" ",
			valueStyle.Render(value),
		)
	}

	b.WriteString(pair("ID", p.Session.ShortID()))
	b.WriteString("\n")

	fullTitle := p.Session.Title
	if len(fullTitle) > 40 {
		fullTitle = fullTitle[:37] + "..."
	}
	b.WriteString(pair("Title", fullTitle))
	b.WriteString("\n")

	dir := truncDir(p.Session.Directory, 35)
	b.WriteString(pair("Dir", dir))
	b.WriteString("\n\n")

	b.WriteString(pair("Created", p.Session.CreatedAt.Format("2006-01-02 15:04")))
	b.WriteString("\n")
	b.WriteString(pair("Updated", p.Session.UpdatedAt.Format("2006-01-02 15:04")))
	b.WriteString("\n")
	b.WriteString(pair("Age", relativeTime(p.Session.UpdatedAt)))
	b.WriteString("\n\n")

	b.WriteString(pair("Messages", fmt.Sprintf("%d", p.MessageCount)))
	b.WriteString("\n\n")

	var status string
	if p.IsBookmark && p.TmuxActive {
		status = "★● Active & Bookmarked"
	} else if p.IsBookmark {
		status = "★ Bookmarked"
	} else if p.TmuxActive {
		status = "● Active in tmux"
	} else {
		status = "○ Inactive"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(p.Theme.TextMuted).
		MarginTop(1)

	b.WriteString(pair("Status", ""))
	b.WriteString("\n")
	b.WriteString(statusStyle.Render("  " + status))

	b.WriteString("\n\n")
	b.WriteString(p.renderHelp())

	return lipgloss.NewStyle().
		Foreground(p.Theme.Text).
		Width(p.Width).
		Height(p.Height).
		Padding(1, 2).
		Render(b.String())
}

func (p *Preview) emptyView() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(p.Theme.TextMuted).
		Italic(true).
		Width(p.Width).
		Height(p.Height).
		Align(lipgloss.Center, lipgloss.Center)

	return emptyStyle.Render("No session selected\n\nUse ↑/↓ to navigate\nEnter to open")
}

func (p *Preview) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(p.Theme.TextMuted).
		MarginTop(1)

	return helpStyle.Render("─ Press Enter to open ─")
}
