package tui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	busyTickInterval   = 50 * time.Millisecond
	busyShowElapsed    = 3 * time.Second
	reduceMotionFrames = 20 // ~1s pulse at 50ms ticks
)

type busyTickMsg time.Time

// spinnerFrameRunes cycles forward then backward like openclaude3 SpinnerGlyph.
var spinnerFrameRunes []rune

func init() {
	base := []rune("·✢✳✶✻✽")
	if os.Getenv("TERM") == "xterm-ghostty" {
		base = []rune("·✢✳✶✻*")
	} else if runtime.GOOS != "darwin" {
		base = []rune("·✢*✶✻✽")
	}
	rev := slices.Clone(base)
	slices.Reverse(rev)
	spinnerFrameRunes = append(append([]rune{}, base...), rev...)
}

func nextBusyTick() tea.Cmd {
	return tea.Tick(busyTickInterval, func(t time.Time) tea.Msg {
		return busyTickMsg(t)
	})
}

func reduceMotionFromEnv() bool {
	for _, key := range []string{"OPENCLAUDE_TUI_REDUCE_MOTION", "ACCESSIBILITY_REDUCED_MOTION"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if v == "1" || v == "true" || v == "yes" {
			return true
		}
	}
	return false
}

func pickSpinnerVerb() string {
	verbs := parsedSpinnerVerbs
	if len(verbs) == 0 {
		return "Working"
	}
	return verbs[rand.N(len(verbs))]
}

func formatBusyElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%d:%02d", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func joinBusyParts(parts ...string) string {
	sep := " · "
	var b strings.Builder
	first := true
	for _, p := range parts {
		if strings.TrimSpace(stripANSI(p)) == "" {
			continue
		}
		if !first {
			b.WriteString(sep)
		}
		first = false
		b.WriteString(p)
	}
	return b.String()
}

// stripANSI removes common SGR sequences for empty-check only.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	in := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			in = true
			i++
			continue
		}
		if in {
			if s[i] == 'm' {
				in = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func (m *model) glimmerRunes(s string, frame int) string {
	rr := []rune(s)
	if len(rr) == 0 {
		return ""
	}
	if m.busyReduceMotion {
		return dimStyle.Render(s)
	}
	idx := (frame / 2) % len(rr)
	var b strings.Builder
	for i, r := range rr {
		if i == idx {
			b.WriteString(userStyle.Render(string(r)))
		} else {
			b.WriteString(dimStyle.Render(string(r)))
		}
	}
	return b.String()
}

func (m *model) renderSpinnerGlyph() string {
	if len(spinnerFrameRunes) == 0 {
		return promptCharStyle.Render("·")
	}
	if m.busyReduceMotion {
		dim := (m.busyFrame/reduceMotionFrames)%2 == 1
		if dim {
			return dimStyle.Render("●")
		}
		return promptCharStyle.Render("●")
	}
	r := spinnerFrameRunes[m.busyFrame%len(spinnerFrameRunes)]
	return promptCharStyle.Render(string(r))
}

func (m *model) renderBusyAnimationLine() string {
	maxW := m.width
	if maxW < 12 {
		maxW = 80
	}

	spin := m.renderSpinnerGlyph()
	verb := m.busyVerb
	if verb == "" {
		verb = "Working"
	}
	ellipsis := "…"

	var thinkingSeg string
	if m.busy && m.runningTool == "" && !m.seenAsstDelta {
		thinkingSeg = m.glimmerRunes("thinking", m.busyFrame+11)
	}

	var toolSeg string
	if m.runningTool != "" {
		toolSeg = toolStyle.Render(m.runningTool)
	}

	var elapsedSeg string
	if !m.busyStart.IsZero() {
		if elapsed := time.Since(m.busyStart); elapsed >= busyShowElapsed {
			elapsedSeg = dimStyle.Render(formatBusyElapsed(elapsed))
		}
	}

	rv := []rune(verb)
	for len(rv) >= 3 {
		msg := m.glimmerRunes(string(rv)+ellipsis, m.busyFrame)
		parts := []string{spin, msg}
		if thinkingSeg != "" {
			parts = append(parts, thinkingSeg)
		}
		if toolSeg != "" {
			parts = append(parts, toolSeg)
		}
		if elapsedSeg != "" {
			parts = append(parts, elapsedSeg)
		}
		line := joinBusyParts(parts...)
		if lipgloss.Width(line) <= maxW {
			return line
		}
		rv = rv[:len(rv)-1]
	}
	return joinBusyParts(spin, dimStyle.Render("…"))
}
