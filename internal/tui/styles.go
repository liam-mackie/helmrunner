package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		PaddingBottom(1)

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("120"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("120"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
)
