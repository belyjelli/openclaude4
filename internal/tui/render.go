package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// renderAssistantMarkdown renders assistant markdown for the terminal; returns "" when disabled or empty.
// themeStyle is "light" or "dark" (from ThemeHolder.MarkdownStyle); drives Chroma palette and contrast.
func renderAssistantMarkdown(width int, text string, enabled bool, themeStyle string) string {
	text = strings.TrimSpace(text)
	if text == "" || !enabled {
		return ""
	}
	style := strings.ToLower(strings.TrimSpace(themeStyle))
	dark := style != "light"
	return renderAssistantMarkdownChroma(width, text, dark, true)
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
// When maxLines > 0, output is capped to that many lines and a dim hint is appended if truncated.
func formatToolResultBody(maxRunes int, maxLines int, body string, width int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var out string
	if looksLikeDiff(body) {
		body = truncateRunes(body, maxRunes)
		out = styleDiffText(body, width)
	} else {
		out = truncateRunes(body, maxRunes)
	}
	return limitToolOutputLines(out, maxLines)
}

// limitToolOutputLines keeps the first maxLines newline-separated rows; maxLines <= 0 means no limit.
func limitToolOutputLines(s string, maxLines int) string {
	s = strings.TrimRight(s, "\n")
	if maxLines <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	omitted := len(lines) - maxLines
	prefix := strings.Join(lines[:maxLines], "\n")
	hint := fmt.Sprintf("… (%d more line(s); raise OPENCLAUDE_TUI_TOOL_MAX_LINES or OPENCLAUDE_TUI_TOOL_PREVIEW)", omitted)
	return prefix + "\n" + dimStyle.Render(hint)
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
