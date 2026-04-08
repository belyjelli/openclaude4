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

	// Rich prompt row (v3 PromptInput-style): rounded frame around ❯ + text field.
	promptBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)
	promptCharStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // v3 user / pointer line
	promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Slash typeahead (v3 PromptInputFooterSuggestions: pointer + full-width selected bar).
	slashRowPrefixSelected = "❯ "
	slashRowPrefixIdle     = "  "
	slashSelectedRowStyle  = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Bold(true)
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
		promptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("252")).
			Background(lipgloss.Color("254")).
			Padding(0, 1)
		promptCharStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("130"))
		promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		slashSelectedRowStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("105")).
			Foreground(lipgloss.Color("235")).
			Bold(true)
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
		promptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)
		promptCharStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		promptCharBusyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		slashSelectedRowStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("255")).
			Bold(true)
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
