package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/providers"
)

func (m *model) applySuggestCompletion() {
	if len(m.slashMatches) == 0 {
		return
	}
	e := m.slashMatches[m.slashSel]
	val := m.ti.Value()
	switch m.compMode {
	case compSlashArg, compFile, compSkill, compMCPResource:
		rs, re := m.replaceStart, m.replaceEnd
		if rs < 0 || re < rs || rs > len(val) {
			return
		}
		if re > len(val) {
			re = len(val)
		}
		newVal := val[:rs] + e.primary + val[re:]
		m.ti.SetValue(newVal)
		m.ti.SetCursor(rs + len(e.primary))
	case compSlashTop:
		leading, trimmed := slashLeadingTrim(val)
		rest := ""
		if i := strings.IndexByte(trimmed, ' '); i >= 0 {
			rest = trimmed[i:]
		}
		repl := "/" + e.primary
		newInner := repl + rest
		newVal := val[:leading] + newInner
		m.ti.SetValue(newVal)
		m.ti.SetCursor(leading + len(newInner))
	default:
		return
	}
	m.slashEscDismiss = false
	m.syncSuggestOverlay()
}

const slashSuggestMaxRows = 4

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
	{primary: "provider", hint: "show | wizard | openai|ollama|gemini|github|openrouter"},
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

func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// overlaySlashOnViewport draws slash suggestions on top of the bottom rows of the viewport string.
// viewportH must match bubbles viewport height so the combined block stays one transcript tile tall.
func overlaySlashOnViewport(base, overlay string, viewportH int) string {
	if strings.TrimSpace(overlay) == "" || viewportH < 1 {
		return base
	}
	oLines := trimTrailingEmptyLines(strings.Split(overlay, "\n"))
	if len(oLines) == 0 {
		return base
	}
	if len(oLines) > viewportH {
		oLines = oLines[len(oLines)-viewportH:]
	}
	oh := len(oLines)
	bLines := trimTrailingEmptyLines(strings.Split(base, "\n"))
	for len(bLines) < viewportH {
		bLines = append(bLines, "")
	}
	if len(bLines) > viewportH {
		bLines = bLines[:viewportH]
	}
	out := make([]string, 0, viewportH)
	out = append(out, bLines[:viewportH-oh]...)
	out = append(out, oLines...)
	return strings.Join(out, "\n")
}

func renderSlashSuggestions(width int, matches []slashEntry, selected int, argMode bool) string {
	if len(matches) == 0 || width < 1 {
		return ""
	}
	header := dimStyle.Width(width).Render("Tab complete · Shift+Tab prev · ↑↓ select · Esc hide · Shift+Tab approvals when hidden")
	start, win := visibleSlashWindow(matches, selected)
	rows := make([]string, 0, len(win))
	for i, e := range win {
		global := start + i
		line := e.display()
		if argMode {
			line = e.primary
		}
		if e.hint != "" {
			line += "  " + dimStyle.Render(e.hint)
		}
		if global == selected {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(line)
		}
		rows = append(rows, lipgloss.NewStyle().Width(width).Render(line))
	}
	if len(matches) > slashSuggestMaxRows {
		more := len(matches) - slashSuggestMaxRows
		rows = append(rows, dimStyle.Width(width).Render(fmt.Sprintf("+%d more", more)))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(width - 2).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *model) clearSuggestOverlay() {
	m.slashMatches = nil
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compNone
	m.replaceStart = -1
	m.replaceEnd = -1
}

// fillSlashOverlay sets slash command or first-argument completions for a line starting with "/".
func (m *model) fillSlashOverlay(leading int, trimmed string) {
	m.slashSuggestIsArg = false
	m.compMode = compSlashTop
	m.replaceStart = -1
	m.replaceEnd = -1
	spaceIdx := strings.IndexByte(trimmed, ' ')
	if spaceIdx < 0 {
		first := trimmed
		stem := strings.TrimPrefix(first, "/")
		m.slashMatches = filterSlashEntries(m.slashAll, stem)
		m.clampSlashSel()
		return
	}
	cmdPart := strings.TrimSpace(trimmed[:spaceIdx])
	cmd := strings.TrimPrefix(cmdPart, "/")
	if cmd == "" {
		m.slashMatches = nil
		m.slashSel = 0
		return
	}
	if strings.EqualFold(cmd, "model") {
		m.fillModelSlashArgs(leading, trimmed, spaceIdx)
		return
	}
	subs, ok := slashSubcommands[strings.ToLower(cmd)]
	if !ok || len(subs) == 0 {
		m.slashMatches = nil
		m.slashSel = 0
		return
	}
	ws, we := argWordBoundsInTrimmed(trimmed, spaceIdx)
	stem := strings.ToLower(trimmed[ws:we])
	var buf []slashEntry
	for _, s := range subs {
		lo := strings.ToLower(s)
		if stem == "" || strings.HasPrefix(lo, stem) {
			buf = append(buf, slashEntry{primary: s, hint: cmd})
		}
	}
	m.slashMatches = buf
	m.slashSuggestIsArg = true
	m.compMode = compSlashArg
	m.replaceStart = leading + ws
	m.replaceEnd = leading + we
	m.clampSlashSel()
}

func (m *model) fillModelSlashArgs(leading int, trimmed string, spaceIdx int) {
	ws, we := argWordBoundsInTrimmed(trimmed, spaceIdx)
	stem := strings.ToLower(trimmed[ws:we])
	opts := providers.CachedChatModelIDsForSuggest()
	var buf []slashEntry
	for _, s := range opts {
		lo := strings.ToLower(s)
		if stem == "" || strings.HasPrefix(lo, stem) {
			buf = append(buf, slashEntry{primary: s, hint: "model"})
		}
	}
	m.slashMatches = buf
	m.slashSuggestIsArg = true
	m.compMode = compSlashArg
	m.replaceStart = leading + ws
	m.replaceEnd = leading + we
	m.clampSlashSel()
}

func (m *model) syncSuggestOverlay() {
	if m.perm != nil {
		m.clearSuggestOverlay()
		return
	}
	val := m.ti.Value()
	if m.slashEscDismiss && val == m.slashDismissSnapshot {
		m.clearSuggestOverlay()
		return
	}
	if val != m.slashDismissSnapshot {
		m.slashEscDismiss = false
	}
	leading, trimmed := slashLeadingTrim(val)
	if strings.HasPrefix(trimmed, "/") && !m.busy {
		m.fillSlashOverlay(leading, trimmed)
		return
	}
	if m.compMode == compFile || m.compMode == compSkill || m.compMode == compMCPResource {
		m.refreshFileSkillMCPOverlay(val)
		return
	}
	m.clearSuggestOverlay()
}

func (m *model) refreshFileSkillMCPOverlay(val string) {
	pos := m.ti.Position()
	ts, te, token := tokenAtCursor(val, pos)
	if token == "" {
		m.clearSuggestOverlay()
		return
	}
	if strings.HasPrefix(token, "@mcp:") {
		m.fillMCPResourceOverlay(val, ts, te, token)
		return
	}
	if strings.HasPrefix(token, "@") {
		m.fillSkillOverlay(val, ts, te, token)
		return
	}
	m.fillPathOverlay(val, ts, te, token)
}

func (m *model) fillPathOverlay(val string, ts, te int, token string) {
	matches := pathCompletionMatches(token)
	if len(matches) == 0 {
		m.clearSuggestOverlay()
		return
	}
	var buf []slashEntry
	for _, x := range matches {
		buf = append(buf, slashEntry{primary: x, hint: "path"})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compFile
	m.replaceStart, m.replaceEnd = ts, te
	m.clampSlashSel()
}

func (m *model) fillSkillOverlay(val string, ts, te int, token string) {
	stem := strings.TrimPrefix(token, "@")
	names := m.skillNamesList()
	matches := skillNameMatches(stem, names)
	if len(matches) == 0 {
		m.clearSuggestOverlay()
		return
	}
	var buf []slashEntry
	for _, n := range matches {
		buf = append(buf, slashEntry{primary: "@" + n, hint: "skill"})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compSkill
	m.replaceStart, m.replaceEnd = ts, te
	m.clampSlashSel()
}

func mcpResourceEntryHint(r mcpclient.MCPResource) string {
	label := strings.TrimSpace(r.Title)
	if label == "" {
		label = strings.TrimSpace(r.Name)
	}
	srv := strings.TrimSpace(r.Server)
	if label == "" {
		return srv
	}
	if srv == "" {
		return label
	}
	return srv + " · " + label
}

func (m *model) fillMCPResourceOverlay(val string, ts, te int, token string) {
	if m.cfg.MCPManager == nil {
		m.clearSuggestOverlay()
		return
	}
	stem := strings.TrimPrefix(token, "@mcp:")
	cands := m.cfg.MCPManager.ResourceSuggestCandidates(stem)
	if len(cands) == 0 {
		m.clearSuggestOverlay()
		return
	}
	var buf []slashEntry
	for _, r := range cands {
		buf = append(buf, slashEntry{primary: r.URI, hint: mcpResourceEntryHint(r)})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compMCPResource
	m.replaceStart, m.replaceEnd = ts, te
	m.clampSlashSel()
}

func (m *model) rebuildSlashMatches() {
	m.syncSuggestOverlay()
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

func (m *model) tryExpandNonSlashTab() bool {
	if m.busy || m.perm != nil {
		return false
	}
	val := m.ti.Value()
	if strings.HasPrefix(strings.TrimLeft(val, " \t"), "/") {
		return false
	}
	pos := m.ti.Position()
	ts, te, token := tokenAtCursor(val, pos)
	if token == "" {
		return false
	}
	if strings.HasPrefix(token, "@mcp:") {
		return m.tabExpandMCPResource(val, ts, te, token)
	}
	if strings.HasPrefix(token, "@") {
		return m.tabExpandSkill(val, ts, te, token)
	}
	return m.tabExpandPath(val, ts, te, token)
}

func (m *model) tabExpandPath(val string, ts, te int, token string) bool {
	matches := pathCompletionMatches(token)
	if len(matches) == 0 {
		return false
	}
	if len(matches) == 1 {
		rep := matches[0]
		newVal := val[:ts] + rep + val[te:]
		m.ti.SetValue(newVal)
		m.ti.SetCursor(ts + len(rep))
		return true
	}
	var buf []slashEntry
	for _, x := range matches {
		buf = append(buf, slashEntry{primary: x, hint: "path"})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compFile
	m.replaceStart, m.replaceEnd = ts, te
	m.slashEscDismiss = false
	return true
}

func (m *model) tabExpandSkill(val string, ts, te int, token string) bool {
	stem := strings.TrimPrefix(token, "@")
	matches := skillNameMatches(stem, m.skillNamesList())
	if len(matches) == 0 {
		return false
	}
	if len(matches) == 1 {
		rep := "@" + matches[0]
		newVal := val[:ts] + rep + val[te:]
		m.ti.SetValue(newVal)
		m.ti.SetCursor(ts + len(rep))
		return true
	}
	var buf []slashEntry
	for _, n := range matches {
		buf = append(buf, slashEntry{primary: "@" + n, hint: "skill"})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compSkill
	m.replaceStart, m.replaceEnd = ts, te
	m.slashEscDismiss = false
	return true
}

func (m *model) tabExpandMCPResource(val string, ts, te int, token string) bool {
	if m.cfg.MCPManager == nil {
		return false
	}
	stem := strings.TrimPrefix(token, "@mcp:")
	cands := m.cfg.MCPManager.ResourceSuggestCandidates(stem)
	if len(cands) == 0 {
		return false
	}
	if len(cands) == 1 {
		rep := cands[0].URI
		newVal := val[:ts] + rep + val[te:]
		m.ti.SetValue(newVal)
		m.ti.SetCursor(ts + len(rep))
		return true
	}
	var buf []slashEntry
	for _, r := range cands {
		buf = append(buf, slashEntry{primary: r.URI, hint: mcpResourceEntryHint(r)})
	}
	m.slashMatches = buf
	m.slashSel = 0
	m.slashSuggestIsArg = false
	m.compMode = compMCPResource
	m.replaceStart, m.replaceEnd = ts, te
	m.slashEscDismiss = false
	return true
}
