package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// renderAssistantMarkdown renders assistant markdown for the terminal; returns "" on failure or when disabled.
func renderAssistantMarkdown(width int, text string, enabled bool) string {
	text = strings.TrimSpace(text)
	if text == "" || !enabled {
		return ""
	}
	w := width
	if w < 40 {
		w = 40
	}
	if w > 120 {
		w = 120
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(w),
		glamour.WithStandardStyle("dark"),
	)
	if err != nil {
		return ""
	}
	out, err := r.Render(text)
	if err != nil {
		return ""
	}
	return strings.TrimRight(out, "\n")
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max-3 {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

func looksLikeDiff(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 3 {
		return false
	}
	adds, dels, at := 0, 0, 0
	for _, ln := range lines {
		if strings.HasPrefix(ln, "@@") {
			at++
		}
		if strings.HasPrefix(ln, "+") && !strings.HasPrefix(ln, "+++") {
			adds++
		}
		if strings.HasPrefix(ln, "-") && !strings.HasPrefix(ln, "---") {
			dels++
		}
	}
	return at >= 1 && adds >= 1 && dels >= 1
}

// formatToolResultBody trims and optionally styles diff-like output; truncates to maxRunes (UTF-8 runes) when not a diff.
func formatToolResultBody(maxRunes int, body string, width int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if looksLikeDiff(body) {
		body = truncateRunes(body, maxRunes)
		return styleDiffText(body, width)
	}
	return truncateRunes(body, maxRunes)
}

func styleDiffText(s string, width int) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	diffPlus := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	diffMinus := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	diffHunk := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styledW := width
	if styledW < 20 {
		styledW = 80
	}
	for _, ln := range lines {
		var line string
		switch {
		case strings.HasPrefix(ln, "@@"):
			line = diffHunk.Width(styledW).Render(ln)
		case strings.HasPrefix(ln, "+") && !strings.HasPrefix(ln, "+++"):
			line = diffPlus.Width(styledW).Render(ln)
		case strings.HasPrefix(ln, "-") && !strings.HasPrefix(ln, "---"):
			line = diffMinus.Width(styledW).Render(ln)
		default:
			line = dim.Width(styledW).Render(ln)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
