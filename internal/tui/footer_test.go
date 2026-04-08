package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	sdk "github.com/sashabaranov/go-openai"
)

func TestHorizontalRuleWidth(t *testing.T) {
	t.Parallel()
	for _, w := range []int{0, 1, 20, 80, 120} {
		s := horizontalRule(w)
		if w < 1 {
			if s != "" {
				t.Fatalf("w=%d: want empty, got %q", w, s)
			}
			continue
		}
		if lipgloss.Width(s) != w {
			t.Fatalf("w=%d: lipgloss width got %d", w, lipgloss.Width(s))
		}
		if len(s) != w {
			t.Fatalf("w=%d: rune len got %d", w, len([]rune(s)))
		}
	}
}

func TestBuildCompactMeterRight(t *testing.T) {
	t.Parallel()
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: strings.Repeat("x", 400)},
	}
	ptr := &msgs
	if s := buildCompactMeterRight(nil, 1000); s != "" {
		t.Fatalf("nil msgs: %q", s)
	}
	if s := buildCompactMeterRight(ptr, 0); s != "" {
		t.Fatalf("threshold 0: %q", s)
	}
	s := buildCompactMeterRight(ptr, 10000)
	if s == "" || !strings.Contains(s, "until auto-compact") {
		t.Fatalf("unexpected: %q", s)
	}
}

func TestFormatFooterRowWidths(t *testing.T) {
	t.Parallel()
	left := "⏵⏵ accept edits on (shift+tab to cycle)"
	right := "42% until auto-compact"
	for _, w := range []int{20, 40, 80, 120} {
		row := formatFooterRow(left, right, w)
		got := lipgloss.Width(row)
		if got > w {
			t.Fatalf("totalWidth=%d: rendered width %d too wide", w, got)
		}
	}
}

func TestFormatFooterRowNarrowPrefersRight(t *testing.T) {
	t.Parallel()
	left := strings.Repeat("a", 100)
	right := "9% until auto-compact"
	w := 30
	row := formatFooterRow(left, right, w)
	if !strings.Contains(row, "until auto-compact") {
		t.Fatalf("expected right side preserved: %q", row)
	}
}

func TestBuildFooterLeftMCP(t *testing.T) {
	t.Parallel()
	s := buildFooterLeft(true, nil)
	if !strings.Contains(s, "accept edits") {
		t.Fatalf("%q", s)
	}
	s2 := buildFooterLeft(false, nil)
	if !strings.Contains(s2, "prompt for approvals") {
		t.Fatalf("%q", s2)
	}
}
