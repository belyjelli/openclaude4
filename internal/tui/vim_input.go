package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) vimMoveRune(delta int) {
	v := m.ti.Value()
	pos := m.ti.Position()
	r := []rune(v)
	if len(r) == 0 {
		return
	}
	newPos := pos + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos > len(r) {
		newPos = len(r)
	}
	m.ti.SetCursor(newPos)
}

func (m *model) vimDeleteChar() {
	v := m.ti.Value()
	pos := m.ti.Position()
	r := []rune(v)
	if pos >= len(r) {
		return
	}
	r = append(r[:pos], r[pos+1:]...)
	m.ti.SetValue(string(r))
	m.ti.SetCursor(pos)
}

// handleVimNormalKey handles keys in vim normal mode on the prompt (consumes all keys).
func (m *model) handleVimNormalKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyLeft:
		m.vimMoveRune(-1)
		return
	case tea.KeyRight:
		m.vimMoveRune(1)
		return
	case tea.KeyHome:
		m.ti.CursorStart()
		return
	case tea.KeyEnd:
		m.ti.CursorEnd()
		return
	case tea.KeyBackspace, tea.KeyDelete, tea.KeyCtrlU, tea.KeyCtrlW:
		return
	case tea.KeySpace:
		return
	case tea.KeyRunes:
		if len(msg.Runes) != 1 {
			return
		}
		switch msg.Runes[0] {
		case 'h':
			m.vimMoveRune(-1)
		case 'l':
			m.vimMoveRune(1)
		case '0':
			m.ti.CursorStart()
		case 'x':
			m.vimDeleteChar()
		case 'i':
			m.vimNormal = false
		case 'I':
			m.ti.CursorStart()
			m.vimNormal = false
		case 'a':
			m.vimMoveRune(1)
			m.vimNormal = false
		case 'A':
			m.ti.CursorEnd()
			m.vimNormal = false
		}
		return
	}

	if msg.String() == "^" {
		m.ti.CursorStart()
		return
	}
	if msg.String() == "$" {
		m.ti.CursorEnd()
	}
}
