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
	permPanelReserveLines = 15
	// pwizPanelReserveLines approximates the provider wizard panel (menu + hints + border).
	pwizPanelReserveLines = 16
)

// horizontalRule returns a line of box-drawing characters exactly totalWidth cells wide (clamped).
func horizontalRule(totalWidth int) string {
	if totalWidth < 1 {
		return ""
	}
	return strings.Repeat("─", totalWidth)
}

func autoApproveEnabled(auto *atomic.Bool) bool {
	if auto == nil {
		return false
	}
	return auto.Load()
}

// buildFooterLeft returns the permission-style left segment (plain text before styling).
func buildFooterLeft(autoOn bool, mcp *mcpclient.Manager) string {
	var b strings.Builder
	if autoOn {
		b.WriteString("⏵⏵ accept edits on (shift+tab to cycle)")
	} else {
		b.WriteString("⏸⏸ prompt for approvals (shift+tab to cycle)")
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

// formatFooterRow lays out left and right in at most totalWidth cells (lipgloss-measured).
// Truncates right first to cap its width, then left; falls back to right-only when too narrow.
func formatFooterRow(left, right string, totalWidth int) string {
	if totalWidth < 1 {
		return ""
	}
	const gapMin = 1
	wR := lipgloss.Width(right)
	maxRight := wR
	if maxRight > totalWidth-gapMin {
		maxRight = totalWidth - gapMin
	}
	if maxRight < 0 {
		maxRight = 0
	}
	rightPart := right
	if wR > maxRight {
		rightPart = truncateVisual(right, maxRight)
	}
	wR = lipgloss.Width(rightPart)

	maxLeft := totalWidth - wR - gapMin
	if maxLeft < 1 {
		rp := truncateVisual(rightPart, totalWidth)
		return lipgloss.Place(totalWidth, 1, lipgloss.Right, lipgloss.Top, dimStyle.Render(rp))
	}

	leftPart := left
	if lipgloss.Width(left) > maxLeft {
		leftPart = truncateVisual(left, maxLeft)
	}
	wL := lipgloss.Width(leftPart)
	gap := totalWidth - wL - wR
	if gap < gapMin {
		leftPart = truncateVisual(leftPart, max(1, maxLeft-(gapMin-gap)))
		wL = lipgloss.Width(leftPart)
		gap = totalWidth - wL - wR
		if gap < gapMin {
			gap = gapMin
		}
	}

	spacer := strings.Repeat(" ", gap)
	line := lipgloss.JoinHorizontal(lipgloss.Top, dimStyle.Render(leftPart), spacer, dimStyle.Render(rightPart))
	if lipgloss.Width(line) > totalWidth {
		rp := truncateVisual(right, totalWidth)
		return lipgloss.Place(totalWidth, 1, lipgloss.Right, lipgloss.Top, dimStyle.Render(rp))
	}
	return line
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

