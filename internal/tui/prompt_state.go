package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) syncPlaceholder() {
	var b strings.Builder
	if m.userSubmitCount < 1 {
		b.WriteString("Try: Summarize README in this directory · ")
	}
	b.WriteString("Message… · Enter send · Shift+Tab approvals · ↑↓ history · prefix+↑ match · Tab @paths vs @skills · @mcp: · ? help · /help")
	m.ti.Placeholder = b.String()
}

func (m *model) resetHistoryNavigation() {
	m.historyIdx = -1
	m.historyFilterActive = false
	m.historyFilterMatches = nil
	m.historyFilterPos = 0
}

// resetHistoryNavigationOnEdit clears browse/filter state when the user edits the prompt (keeps input value).
func (m *model) resetHistoryNavigationOnEdit() {
	if !m.historyFilterActive && m.historyIdx < 0 {
		return
	}
	m.resetHistoryNavigation()
}

func (m *model) appendHistory(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == line {
		return
	}
	m.inputHistory = append(m.inputHistory, line)
	const maxHist = 500
	if len(m.inputHistory) > maxHist {
		m.inputHistory = m.inputHistory[len(m.inputHistory)-maxHist:]
		m.resetHistoryNavigation()
	}
}

// historyIndicesMatchingPrefix returns indices into inputHistory (newest first) where the trimmed line
// has a case-insensitive prefix match.
func historyIndicesMatchingPrefix(hist []string, prefix string) []int {
	p := strings.ToLower(strings.TrimSpace(prefix))
	if p == "" {
		return nil
	}
	var out []int
	for i := len(hist) - 1; i >= 0; i-- {
		line := strings.TrimSpace(hist[i])
		if strings.HasPrefix(strings.ToLower(line), p) {
			out = append(out, i)
		}
	}
	return out
}

// historyUp returns (consumed, optionalToastCmd). Prefix + ↑ filters history (shell-style); empty line + ↑ browses all.
func (m *model) historyUp() (bool, tea.Cmd) {
	if len(m.inputHistory) == 0 {
		return false, nil
	}

	if m.historyFilterActive {
		if len(m.historyFilterMatches) == 0 {
			m.resetHistoryNavigation()
			return m.historyUp()
		}
		if m.historyFilterPos < len(m.historyFilterMatches)-1 {
			m.historyFilterPos++
			m.ti.SetValue(m.inputHistory[m.historyFilterMatches[m.historyFilterPos]])
			m.ti.CursorEnd()
			return true, nil
		}
		m.ti.SetValue(m.inputHistory[m.historyFilterMatches[m.historyFilterPos]])
		m.ti.CursorEnd()
		return true, nil
	}

	if m.historyIdx >= 0 {
		if m.historyIdx > 0 {
			m.historyIdx--
			m.ti.SetValue(m.inputHistory[m.historyIdx])
			m.ti.CursorEnd()
			return true, nil
		}
		m.ti.SetValue(m.inputHistory[0])
		m.ti.CursorEnd()
		return true, nil
	}

	prefix := strings.TrimSpace(m.ti.Value())
	if prefix != "" {
		matches := historyIndicesMatchingPrefix(m.inputHistory, prefix)
		if len(matches) == 0 {
			cmd := m.pushToast("no history match", toastWarn)
			return true, cmd
		}
		m.historyDraft = m.ti.Value()
		m.historyFilterActive = true
		m.historyFilterMatches = matches
		m.historyFilterPos = 0
		m.historyIdx = -1
		m.ti.SetValue(m.inputHistory[matches[0]])
		m.ti.CursorEnd()
		return true, nil
	}

	m.historyDraft = m.ti.Value()
	m.historyIdx = len(m.inputHistory) - 1
	m.ti.SetValue(m.inputHistory[m.historyIdx])
	m.ti.CursorEnd()
	return true, nil
}

func (m *model) historyDown() (bool, tea.Cmd) {
	if m.historyFilterActive {
		if len(m.historyFilterMatches) == 0 {
			m.resetHistoryNavigation()
			return false, nil
		}
		if m.historyFilterPos > 0 {
			m.historyFilterPos--
			m.ti.SetValue(m.inputHistory[m.historyFilterMatches[m.historyFilterPos]])
			m.ti.CursorEnd()
			return true, nil
		}
		m.resetHistoryNavigation()
		m.ti.SetValue(m.historyDraft)
		m.ti.CursorEnd()
		return true, nil
	}

	if m.historyIdx < 0 {
		return false, nil
	}
	if m.historyIdx < len(m.inputHistory)-1 {
		m.historyIdx++
		m.ti.SetValue(m.inputHistory[m.historyIdx])
		m.ti.CursorEnd()
		return true, nil
	}
	m.historyIdx = -1
	m.ti.SetValue(m.historyDraft)
	m.ti.CursorEnd()
	return true, nil
}
