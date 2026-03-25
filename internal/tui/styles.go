package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("36")).
			Bold(true).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("36"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	previewTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("36")).
				Bold(true).
				MarginBottom(1)

	previewLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	previewValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).Faint(true)

	favoriteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	idStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("96"))

	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("36")).
			Bold(true).
			MarginTop(1).MarginBottom(1)

	statsValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)
