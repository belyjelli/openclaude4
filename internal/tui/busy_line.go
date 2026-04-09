package tui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/session"
)

const (
	busyTickInterval   = 50 * time.Millisecond
	busyShowElapsed    = 3 * time.Second
	busyShowTokens     = 30 * time.Second // v3 SHOW_TOKENS_AFTER_MS
	reduceMotionFrames = 20              // ~1s pulse at 50ms ticks
	stallQuietAfter    = 3 * time.Second // v3 useStalledAnimation: red after 3s no growth
	stallRampDuration  = 2 * time.Second // v3: full red over 2s after threshold
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

// effectiveSpinnerVerbs merges built-in verbs with config tui.spinner_verbs (v3 parity).
func effectiveSpinnerVerbs() []string {
	replace, extra := config.TUISpinnerVerbConfig()
	if len(extra) == 0 {
		return parsedSpinnerVerbs
	}
	if replace {
		return extra
	}
	out := make([]string, 0, len(parsedSpinnerVerbs)+len(extra))
	out = append(out, parsedSpinnerVerbs...)
	out = append(out, extra...)
	return out
}

func (m *model) pickBusyLineVerb() string {
	if m.cfg.BusySpinnerVerb != nil {
		if v := strings.TrimSpace(m.cfg.BusySpinnerVerb()); v != "" {
			return v
		}
	}
	verbs := effectiveSpinnerVerbs()
	if len(verbs) == 0 {
		return "Working"
	}
	return verbs[rand.N(len(verbs))]
}

func (m *model) smoothStallTowards(target float64) {
	if m.busyReduceMotion {
		m.stallSmoothed = target
		return
	}
	diff := target - m.stallSmoothed
	if diff > -0.01 && diff < 0.01 {
		m.stallSmoothed = target
		return
	}
	m.stallSmoothed += diff * 0.1
}

func (m *model) stallTargetIntensity(now time.Time) float64 {
	if m.runningTool != "" {
		return 0
	}
	var since time.Duration
	if m.liveAsst.Len() == 0 && !m.seenAsstDelta {
		since = now.Sub(m.busyStart)
	} else {
		since = now.Sub(m.lastStreamChange)
	}
	if since <= stallQuietAfter {
		return 0
	}
	excess := float64(since - stallQuietAfter)
	intensity := excess / float64(stallRampDuration)
	if intensity > 1 {
		return 1
	}
	return intensity
}

func formatBusyInt(n int) string {
	if n < 0 {
		n = 0
	}
	s := strconv.Itoa(n)
	if len(s) <= 4 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(s[:lead])
	for i := lead; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func (m *model) showBusyLineTokens() bool {
	if m.cfg.Messages == nil {
		return false
	}
	if config.TUIBusyLineVerboseTokens() {
		return true
	}
	if m.busyStart.IsZero() {
		return false
	}
	return time.Since(m.busyStart) >= busyShowTokens
}

func (m *model) busyLineTokenSegment() string {
	if !m.showBusyLineTokens() {
		return ""
	}
	msgs := *m.cfg.Messages
	n := session.RoughTokenEstimate(msgs)
	// v3 leader: "↓ N tokens"
	return dimStyle.Render("↓ " + formatBusyInt(n) + " tokens")
}

func spinnerStyleForStall(t float64) lipgloss.Style {
	if t <= 0 {
		return promptCharStyle
	}
	if t >= 1 {
		return errStyle
	}
	// Approximate lerp from warm accent (xterm 214-ish) toward err red (xterm 203-ish).
	const (
		loR, loG, loB = 255, 175, 95
		hiR, hiG, hiB = 255, 95, 175
	)
	r := uint8(float64(loR) + t*float64(hiR-loR))
	g := uint8(float64(loG) + t*float64(hiG-loG))
	b := uint8(float64(loB) + t*float64(hiB-loB))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b)))
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
		return spinnerStyleForStall(m.stallSmoothed).Render("·")
	}
	if m.busyReduceMotion {
		dim := (m.busyFrame/reduceMotionFrames)%2 == 1
		st := spinnerStyleForStall(m.stallSmoothed)
		if dim {
			return dimStyle.Render("●")
		}
		return st.Render("●")
	}
	r := spinnerFrameRunes[m.busyFrame%len(spinnerFrameRunes)]
	return spinnerStyleForStall(m.stallSmoothed).Render(string(r))
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

	tokenSeg := m.busyLineTokenSegment()

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
		if tokenSeg != "" {
			parts = append(parts, tokenSeg)
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
