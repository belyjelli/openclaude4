package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	slashSuggestMaxRows     = 4
	slashSuggestHeaderLines = 0 // v3 overlay is rows only (no title bar)
)

// slashEntry is one completable slash command (primary name without leading /).
type slashEntry struct {
	primary string
	aliases []string
	hint    string
}

func (e slashEntry) display() string {
	return "/" + e.primary
}

func (e slashEntry) matchesStem(stem string) bool {
	stem = strings.ToLower(strings.TrimSpace(stem))
	p := strings.ToLower(e.primary)
	if stem == "" {
		return true
	}
	if strings.HasPrefix(p, stem) {
		return true
	}
	for _, a := range e.aliases {
		a = strings.ToLower(a)
		if strings.HasPrefix(a, stem) {
			return true
		}
	}
	return false
}

// staticSlashEntries mirrors cmd/openclaude/slash.go built-ins (first-token completion).
var staticSlashEntries = []slashEntry{
	{primary: "btw", hint: "side question (isolated completion)"},
	{primary: "clear", hint: "clear transcript"},
	{primary: "compact", hint: "lossy tail compaction"},
	{primary: "config", hint: "effective config summary"},
	{primary: "context", aliases: []string{"tokens"}, hint: "message + rough token stats"},
	{primary: "copy", hint: "last assistant → clipboard"},
	{primary: "cost", aliases: []string{"usage"}, hint: "transcript stats"},
	{primary: "doctor", hint: "diagnostics"},
	{primary: "exit", aliases: []string{"quit"}, hint: "leave chat"},
	{primary: "export", hint: "json | md [path]"},
	{primary: "help", hint: "slash help"},
	{primary: "init", hint: "starter yaml snippet"},
	{primary: "mcp", hint: "list | config | doctor | add | help"},
	{primary: "model", hint: "show or set model id"},
	{primary: "onboard", aliases: []string{"setup"}, hint: "onboarding hints"},
	{primary: "permissions", hint: "auto-approve + MCP approval"},
	{primary: "provider", hint: "show | wizard | openai|ollama|gemini|github"},
	{primary: "resume", hint: "list or load session"},
	{primary: "session", hint: "show | list | save | load | new | running | ps"},
	{primary: "skills", hint: "list | read <name>"},
	{primary: "theme", hint: "light | dark | auto (TUI)"},
	{primary: "version", hint: "build version"},
	{primary: "vim", hint: "toggle vim-style prompt (TUI)"},
}

func init() {
	sort.Slice(staticSlashEntries, func(i, j int) bool {
		return strings.ToLower(staticSlashEntries[i].primary) < strings.ToLower(staticSlashEntries[j].primary)
	})
}

func buildSlashIndex(skillNames func() []string) []slashEntry {
	out := make([]slashEntry, len(staticSlashEntries), len(staticSlashEntries)+32)
	copy(out, staticSlashEntries)
	seen := map[string]struct{}{}
	for _, e := range staticSlashEntries {
		seen[strings.ToLower(e.primary)] = struct{}{}
	}
	if skillNames != nil {
		for _, n := range skillNames() {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			low := strings.ToLower(n)
			if _, ok := seen[low]; ok {
				continue
			}
			seen[low] = struct{}{}
			out = append(out, slashEntry{primary: n, hint: "skill"})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].primary) < strings.ToLower(out[j].primary)
	})
	return out
}

func filterSlashEntries(all []slashEntry, stem string) []slashEntry {
	var buf []slashEntry
	for _, e := range all {
		if e.matchesStem(stem) {
			buf = append(buf, e)
		}
	}
	return buf
}

// visibleSlashWindow returns a slice of at most slashSuggestMaxRows entries centered around selected.
func visibleSlashWindow(matches []slashEntry, selected int) (start int, slice []slashEntry) {
	if len(matches) == 0 {
		return 0, nil
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(matches) {
		selected = len(matches) - 1
	}
	if len(matches) <= slashSuggestMaxRows {
		return 0, matches
	}
	start = selected - 1
	if start < 0 {
		start = 0
	}
	if start+slashSuggestMaxRows > len(matches) {
		start = len(matches) - slashSuggestMaxRows
	}
	return start, matches[start : start+slashSuggestMaxRows]
}

func suggestionContentLines(matches []slashEntry) int {
	if len(matches) == 0 {
		return 0
	}
	if len(matches) < slashSuggestMaxRows {
		return len(matches)
	}
	return slashSuggestMaxRows
}

// suggestionBlockHeight is total terminal rows for the slash overlay (0 = hidden).
func suggestionBlockHeight(matches []slashEntry) int {
	if len(matches) == 0 {
		return 0
	}
	h := slashSuggestHeaderLines + suggestionContentLines(matches)
	if len(matches) > slashSuggestMaxRows {
		h++
	}
	return h
}

func slashPrefixColWidth() int {
	a := lipgloss.Width(slashRowPrefixSelected)
	b := lipgloss.Width(slashRowPrefixIdle)
	if a > b {
		return a
	}
	return b
}

func padVisualCells(s string, target int) string {
	if target < 1 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= target {
		return s
	}
	return s + strings.Repeat(" ", target-w)
}

func maxSlashCommandCol(win []slashEntry, totalWidth int) int {
	maxW := 0
	for _, e := range win {
		n := lipgloss.Width("/" + e.primary)
		if n > maxW {
			maxW = n
		}
	}
	capW := totalWidth * 2 / 5
	if capW < 14 {
		capW = 14
	}
	if maxW > capW {
		maxW = capW
	}
	if maxW < 10 {
		maxW = 10
	}
	return maxW
}

func fillRowPlain(s string, width int) string {
	if width < 1 {
		return ""
	}
	for lipgloss.Width(s) < width {
		s += " "
	}
	if lipgloss.Width(s) > width {
		return truncateVisual(s, width)
	}
	return s
}

// renderSlashSuggestions draws v3-style rows: ❯ on the selected line, padded command column, dim hint on the right, full-width inverted bar when selected (no bordered box).
func renderSlashSuggestions(width int, matches []slashEntry, selected int) string {
	if len(matches) == 0 || width < 1 {
		return ""
	}
	prefixW := slashPrefixColWidth()
	start, win := visibleSlashWindow(matches, selected)
	cmdCol := maxSlashCommandCol(win, width)

	rows := make([]string, 0, len(win)+1)
	for i, e := range win {
		global := start + i
		isSel := global == selected
		pre := slashRowPrefixIdle
		if isSel {
			pre = slashRowPrefixSelected
		}
		pre = padVisualCells(pre, prefixW)
		name := truncateVisual("/"+e.primary, cmdCol)
		namePad := padVisualCells(name, cmdCol)
		used := lipgloss.Width(pre) + lipgloss.Width(namePad)
		descW := width - used
		if descW < 1 {
			descW = 0
		}
		hint := strings.TrimSpace(strings.ReplaceAll(e.hint, "\n", " "))
		var desc string
		if hint != "" && descW > 1 {
			desc = " " + truncateVisual(hint, descW-1)
		}
		plain := pre + namePad + desc
		plain = fillRowPlain(plain, width)
		if isSel {
			rows = append(rows, slashSelectedRowStyle.Width(width).Render(plain))
		} else {
			rows = append(rows, dimStyle.Width(width).Render(plain))
		}
	}
	if len(matches) > slashSuggestMaxRows {
		more := len(matches) - slashSuggestMaxRows
		line := fillRowPlain(padVisualCells(slashRowPrefixIdle, prefixW)+fmt.Sprintf("+%d more", more), width)
		rows = append(rows, dimStyle.Width(width).Render(line))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *model) rebuildSlashMatches() {
	if m.perm != nil {
		m.slashMatches = nil
		m.slashSel = 0
		return
	}
	val := m.ti.Value()
	if m.slashEscDismiss && val == m.slashDismissSnapshot {
		m.slashMatches = nil
		return
	}
	if val != m.slashDismissSnapshot {
		m.slashEscDismiss = false
	}
	if !strings.HasPrefix(val, "/") || m.busy {
		m.slashMatches = nil
		m.slashSel = 0
		return
	}
	first := val
	if i := strings.IndexByte(val, ' '); i >= 0 {
		first = val[:i]
	}
	stem := strings.TrimPrefix(first, "/")
	m.slashMatches = filterSlashEntries(m.slashAll, stem)
	m.clampSlashSel()
}

func (m *model) clampSlashSel() {
	if len(m.slashMatches) == 0 {
		m.slashSel = 0
		return
	}
	if m.slashSel >= len(m.slashMatches) {
		m.slashSel = len(m.slashMatches) - 1
	}
	if m.slashSel < 0 {
		m.slashSel = 0
	}
}

func (m *model) applySlashCompletion() {
	if len(m.slashMatches) == 0 {
		return
	}
	e := m.slashMatches[m.slashSel]
	val := m.ti.Value()
	rest := ""
	if i := strings.IndexByte(val, ' '); i >= 0 {
		rest = val[i:]
	}
	repl := "/" + e.primary
	m.ti.SetValue(repl + rest)
	m.ti.SetCursor(len(repl))
	m.slashEscDismiss = false
	m.slashSel = 0
	m.rebuildSlashMatches()
}
