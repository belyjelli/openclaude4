package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	toastNone = iota
	toastInfo
	toastWarn
	toastErr
)

type toastClearMsg struct {
	id int
}

func (m *model) pushToast(text string, kind int) tea.Cmd {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	w := m.width
	if w < 1 {
		w = 80
	}
	if lipgloss.Width(text) > w {
		text = truncateVisual(text, w)
	}
	m.toastText = text
	m.toastKind = kind
	m.toastClearID++
	id := m.toastClearID
	m.reflowLayout()
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{id: id}
	})
}

func (m *model) renderToastLine() string {
	if strings.TrimSpace(m.toastText) == "" {
		return ""
	}
	w := m.width
	if w < 1 {
		w = 80
	}
	// v3-style notification strip: inset like fullscreen PromptInput (paddingLeft≈2), single row.
	pad := 2
	avail := w - pad
	if avail < 4 {
		avail = w - 1
		if avail < 1 {
			avail = 1
		}
		pad = w - avail
	}
	text := m.toastText
	if lipgloss.Width(text) > avail {
		text = truncateVisual(text, avail)
	}
	prefix := strings.Repeat(" ", pad)
	plain := prefix + text
	switch m.toastKind {
	case toastErr:
		st := lipgloss.NewStyle().Width(w).Foreground(lipgloss.Color("224"))
		if lipgloss.HasDarkBackground() {
			st = st.Background(lipgloss.Color("52"))
		} else {
			st = st.Background(lipgloss.Color("224")).Foreground(lipgloss.Color("52"))
		}
		return st.Render(fillToastRow(plain, w))
	case toastWarn:
		st := lipgloss.NewStyle().Width(w).Foreground(lipgloss.Color("230"))
		if lipgloss.HasDarkBackground() {
			st = st.Background(lipgloss.Color("94"))
		} else {
			st = st.Background(lipgloss.Color("229")).Foreground(lipgloss.Color("94"))
		}
		return st.Render(fillToastRow(plain, w))
	default:
		return dimStyle.Width(w).Render(fillToastRow(plain, w))
	}
}

func fillToastRow(s string, w int) string {
	for lipgloss.Width(s) < w {
		s += " "
	}
	if lipgloss.Width(s) > w {
		return truncateVisual(s, w)
	}
	return s
}
