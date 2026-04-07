package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	userStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	asstStyle  = lipgloss.NewStyle()
	toolStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	border     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)
