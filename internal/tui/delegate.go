package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/adruban/osem/internal/tui/theme"
)

type itemDelegate struct {
	theme theme.Theme
}

func newItemDelegate(t theme.Theme) itemDelegate {
	return itemDelegate{theme: t}
}

func (d itemDelegate) Height() int {
	return 2
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(sessionItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	var sb strings.Builder

	// Status indicator
	var status string
	if i.isBookmark && i.isTmuxActive {
		status = "★●"
	} else if i.isBookmark {
		status = "★ "
	} else if i.isTmuxActive {
		status = "● "
	} else {
		status = "  "
	}

	// Title line
	titleStyle := lipgloss.NewStyle()
	if isSelected {
		titleStyle = titleStyle.
			Foreground(d.theme.TextBright).
			Bold(true).
			Background(d.theme.SurfaceHighlight)
	} else {
		titleStyle = titleStyle.Foreground(d.theme.Text)
	}

	title := i.session.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	sb.WriteString(status)
	sb.WriteString(" ")
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n")

	// Description line: ID · Dir · Age
	idStyle := lipgloss.NewStyle().Foreground(d.theme.Primary)
	dirStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted)
	ageStyle := lipgloss.NewStyle().Foreground(d.theme.Secondary)

	shortID := i.session.ShortID()
	dir := truncDir(i.session.Directory, 30)
	age := relativeTime(i.session.UpdatedAt)

	var descLine string
	if isSelected {
		descLine = fmt.Sprintf("  %s · %s · %s",
			idStyle.Render(shortID),
			dirStyle.Render(dir),
			ageStyle.Render(age),
		)
		descLine = lipgloss.NewStyle().
			Background(d.theme.SurfaceHighlight).
			Foreground(d.theme.Text).
			Render(descLine)
	} else {
		descLine = fmt.Sprintf("  %s · %s · %s",
			idStyle.Render(shortID),
			dirStyle.Render(dir),
			ageStyle.Render(age),
		)
	}

	sb.WriteString(descLine)

	// Render with selection highlight
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Background(d.theme.SurfaceHighlight).
			Width(m.Width()-4).
			Padding(0, 1)

		fmt.Fprint(w, selectedStyle.Render(sb.String()))
	} else {
		normalStyle := lipgloss.NewStyle().
			Width(m.Width()-4).
			Padding(0, 1)

		fmt.Fprint(w, normalStyle.Render(sb.String()))
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}

func truncDir(dir string, maxLen int) string {
	if len(dir) <= maxLen {
		return dir
	}
	if strings.HasPrefix(dir, "/home/") {
		parts := strings.SplitN(dir, "/", 4)
		if len(parts) >= 4 {
			short := "~/" + parts[3]
			if len(short) <= maxLen {
				return short
			}
			return "..." + short[len(short)-maxLen+3:]
		}
	}
	return "..." + dir[len(dir)-maxLen+3:]
}
