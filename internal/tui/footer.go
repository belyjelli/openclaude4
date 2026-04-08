package tui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/session"
	sdk "github.com/sashabaranov/go-openai"
)

const (
	// promptChromeLines is the exact row count below the transcript for: top rule, input, bottom rule, footer hint.
	promptChromeLines = 4
	// permPanelReserveLines approximates the permission modal height for viewport budgeting.
	permPanelReserveLines = 10
)

// horizontalRule returns a line of box-drawing characters exactly totalWidth cells wide (clamped).
func horizontalRule(totalWidth int) string {
	if totalWidth < 1 {
		return ""
	}
	return strings.Repeat("─", totalWidth)
}

func autoApproveEnabled(auto *atomic.Bool, legacy bool) bool {
	if auto != nil {
		return auto.Load()
	}
	return legacy
}

// buildFooterLeft returns the permission-style left segment (plain text before styling).
func buildFooterLeft(autoOn bool, mcp *mcpclient.Manager) string {
	var b strings.Builder
	if autoOn {
		b.WriteString("⏵⏵ accept edits on (shift+tab to cycle)")
	} else {
		b.WriteString("⏸⏸ default on (shift+tab to cycle)")
	}
	if suf := mcpNonAskSummary(mcp); suf != "" {
		b.WriteString(" · ")
		b.WriteString(suf)
	}
	return b.String()
}

func mcpNonAskSummary(mcp *mcpclient.Manager) string {
	if mcp == nil || len(mcp.Servers) == 0 {
		return ""
	}
	var parts []string
	for _, s := range mcp.Servers {
		a := strings.ToLower(strings.TrimSpace(s.Approval()))
		if a != "" && a != "ask" {
			parts = append(parts, fmt.Sprintf("%s=%s", s.Name, a))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	const maxLen = 48
	s := "mcp: " + strings.Join(parts, ", ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// buildCompactMeterRight returns the right footer segment for auto-compact headroom, or empty if disabled.
// Mirrors v3-style "N% until auto-compact": percent of threshold budget not yet consumed (rough estimate).
func buildCompactMeterRight(msgs *[]sdk.ChatCompletionMessage, threshold int) string {
	if threshold <= 0 || msgs == nil {
		return ""
	}
	est := session.RoughTokenEstimate(*msgs)
	// Headroom fraction: how much of the threshold budget remains before compaction triggers.
	// displayPercentLeft in v3 TokenWarning uses similar "until auto-compact" wording.
	pct := (100 * (threshold - est)) / threshold
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return fmt.Sprintf("%d%% until auto-compact", pct)
}

// formatFooterRow lays out left and right strings in exactly totalWidth terminal columns using lipgloss.
// On very narrow widths, truncates the left side first so the right (compact meter) can remain visible.
func formatFooterRow(left, right string, totalWidth int) string {
	if totalWidth < 1 {
		return ""
	}
	wLeft := lipgloss.Width(left)
	wRight := lipgloss.Width(right)
	gap := totalWidth - wLeft - wRight
	if gap < 1 {
		// Reserve at least a minimal gap, truncate left to fit right + "  "
		priorityRight := right
		wPR := lipgloss.Width(priorityRight)
		maxLeft := totalWidth - wPR - 1
		if maxLeft < 4 {
			// Too narrow: stack is awkward; show right only if non-empty else left truncated
			if priorityRight != "" {
				left = ""
				return lipgloss.Place(totalWidth, 1, lipgloss.Right, lipgloss.Top, dimStyle.Render(priorityRight))
			}
			left = truncateRunes(left, max(1, totalWidth-1))
			return lipgloss.Place(totalWidth, 1, lipgloss.Left, lipgloss.Top, dimStyle.Render(left))
		}
		left = truncateVisual(left, maxLeft)
		wLeft = lipgloss.Width(left)
		gap = totalWidth - wLeft - wPR
		if gap < 1 {
			gap = 1
		}
	}
	spacer := strings.Repeat(" ", gap)
	return lipgloss.JoinHorizontal(lipgloss.Top, dimStyle.Render(left), spacer, dimStyle.Render(right))
}

func truncateVisual(s string, maxCells int) string {
	if maxCells < 1 {
		return ""
	}
	if lipgloss.Width(s) <= maxCells {
		return s
	}
	if maxCells <= 1 {
		return "…"
	}
	target := maxCells - lipgloss.Width("…")
	if target < 1 {
		return "…"
	}
	for len(s) > 0 && lipgloss.Width(s) > target {
		_, size := utf8.DecodeLastRuneInString(s)
		if size == 0 {
			break
		}
		s = s[:len(s)-size]
	}
	return s + "…"
}

func truncateRunes(s string, maxCells int) string {
	if maxCells < 1 {
		return ""
	}
	if lipgloss.Width(s) <= maxCells {
		return s
	}
	for len(s) > 0 && lipgloss.Width(s) > maxCells {
		_, size := utf8.DecodeLastRuneInString(s)
		if size == 0 {
			break
		}
		s = s[:len(s)-size]
	}
	return s
}
