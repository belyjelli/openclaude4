package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// ApplyTheme updates lipgloss palette presets. Mode is light, dark, or auto (uses terminal background).
func ApplyTheme(mode string) {
	switch resolveThemeMode(mode) {
	case "light":
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("25"))
		userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("130"))
		asstStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("235"))
		toolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("56"))
		okStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("28"))
		errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("160"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Foreground(lipgloss.Color("240"))
	default:
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		asstStyle = lipgloss.NewStyle()
		toolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
		okStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	}
}

func resolveThemeMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "auto" || m == "" {
		if lipgloss.HasDarkBackground() {
			return "dark"
		}
		return "light"
	}
	if m == "light" || m == "dark" {
		return m
	}
	return "dark"
}
