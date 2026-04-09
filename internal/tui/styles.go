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
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	border     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

	// Prompt input row: full width, no frame (chrome is one line above + footer below).
	promptRowStyle = lipgloss.NewStyle()
	promptCharStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // v3 user / pointer line
	promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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
		warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("130"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Foreground(lipgloss.Color("240"))
		promptRowStyle = lipgloss.NewStyle()
		promptCharStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("130"))
		promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	default:
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		asstStyle = lipgloss.NewStyle()
		toolStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
		okStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		promptRowStyle = lipgloss.NewStyle()
		promptCharStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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
