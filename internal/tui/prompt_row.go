package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// promptPointer matches v3 Ink figures.pointer (❯) beside the text field.
const promptPointer = "❯"

func (m *model) promptCharRendered() string {
	s := promptPointer + " "
	if m.busy {
		return promptCharBusyStyle.Render(s)
	}
	return promptCharStyle.Render(s)
}

// promptPrefixWidth is the horizontal cells used inside the prompt box before the bubbles text field.
func (m *model) promptPrefixWidth() int {
	n := 0
	if m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() {
		if m.vimNormal {
			n += lipgloss.Width(dimStyle.Render("(vim NOR) "))
		} else {
			n += lipgloss.Width(dimStyle.Render("(vim INS) "))
		}
	}
	n += lipgloss.Width(m.promptCharRendered())
	return n
}

// textInputWidth returns bubbles textinput width: full row minus rounded border, padding, and prefix (v3-style chrome).
func (m *model) textInputWidth() int {
	if m.width <= 0 {
		return 20
	}
	frame := promptBoxStyle.GetHorizontalFrameSize()
	inner := m.width - frame
	w := inner - m.promptPrefixWidth()
	return max(1, w)
}
