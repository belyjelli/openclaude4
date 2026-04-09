package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// compMode selects how Tab applies the selected suggestion.
const (
	compNone = iota
	compSlashTop
	compSlashArg
	compFile
	compSkill
	compMCPResource
)

// slashSubcommands are first-token completions after "/<cmd> ".
var slashSubcommands = map[string][]string{
	"session":  {"show", "list", "save", "load", "new", "running", "ps"},
	"mcp":      {"list", "config", "doctor", "add", "help"},
	"skills":   {"list", "read"},
	"export":   {"json", "md"},
	"provider": {"show", "wizard", "openai", "ollama", "gemini", "github", "openrouter"},
	"theme":    {"light", "dark", "auto"},
	"resume":   {"list", "load"},
}

func slashLeadingTrim(val string) (leading int, trimmed string) {
	for leading = 0; leading < len(val); leading++ {
		c := val[leading]
		if c != ' ' && c != '\t' {
			break
		}
	}
	return leading, val[leading:]
}

func argWordBoundsInTrimmed(trimmed string, spaceIdx int) (start, end int) {
	if spaceIdx < 0 || spaceIdx >= len(trimmed) {
		return len(trimmed), len(trimmed)
	}
	rest := trimmed[spaceIdx+1:]
	trimRest := strings.TrimLeft(rest, " \t")
	if trimRest == "" {
		p := spaceIdx + 1 + len(rest)
		return p, p
	}
	delta := len(rest) - len(trimRest)
	start = spaceIdx + 1 + delta
	end = start
	for end < len(trimmed) && !unicode.IsSpace(rune(trimmed[end])) {
		end++
	}
	return start, end
}

// tokenAtCursor returns byte offsets [start,end) and the token substring under pos in val.
func tokenAtCursor(val string, pos int) (start, end int, token string) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(val) {
		pos = len(val)
	}
	isSep := func(b byte) bool {
		return b == ' ' || b == '\t' || b == '"' || b == '\''
	}
	start = pos
	for start > 0 && !isSep(val[start-1]) {
		start--
	}
	end = pos
	for end < len(val) && !isSep(val[end]) {
		end++
	}
	if start <= end {
		token = val[start:end]
	}
	return start, end, token
}

func pathCompletionMatches(token string) []string {
	if token == "" {
		return nil
	}
	dir, base := filepath.Split(token)
	listDir := dir
	if listDir == "" {
		listDir = "."
	}
	if strings.HasSuffix(token, string(filepath.Separator)) {
		listDir = filepath.Clean(token)
		base = ""
	} else {
		listDir = filepath.Clean(listDir)
		if listDir == "." && dir == "" {
			listDir = "."
		}
	}
	entries, err := os.ReadDir(listDir)
	if err != nil {
		return nil
	}
	var out []string
	prefix := strings.ToLower(base)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			joined := filepath.Join(dir, name)
			if e.IsDir() {
				joined += string(filepath.Separator)
			}
			out = append(out, joined)
		}
	}
	sort.Strings(out)
	return out
}

func skillNameMatches(stem string, names []string) []string {
	stem = strings.ToLower(strings.TrimSpace(stem))
	var out []string
	seen := map[string]struct{}{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		low := strings.ToLower(n)
		if _, ok := seen[low]; ok {
			continue
		}
		if stem == "" || strings.HasPrefix(low, stem) {
			seen[low] = struct{}{}
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func (m *model) skillNamesList() []string {
	if m.cfg.SkillNames == nil {
		return nil
	}
	return m.cfg.SkillNames()
}

// tryQuestionMarkHelp runs /help when user types ? at an empty prompt (v3-style).
func (m *model) tryQuestionMarkHelp(msg tea.KeyMsg) bool {
	if m.busy || m.perm != nil || m.slashSuggestActive() {
		return false
	}
	if m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() && m.vimNormal {
		return false
	}
	if msg.Type != tea.KeyRunes || len(msg.Runes) != 1 || msg.Runes[0] != '?' {
		return false
	}
	if strings.TrimSpace(m.ti.Value()) != "" || m.ti.Position() != 0 {
		return false
	}
	if m.cfg.Slash == nil {
		return false
	}
	out, exit, err := m.cfg.Slash("/help")
	if exit {
		return false
	}
	if err != nil {
		m.commitLine(errStyle.Render("Error: ") + err.Error())
		return true
	}
	if strings.TrimSpace(out) != "" {
		m.commitLine(out)
	}
	return true
}
