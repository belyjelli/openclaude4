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
	line := m.toastText
	switch m.toastKind {
	case toastErr:
		return errStyle.Width(w).Render(line)
	case toastWarn:
		return warnStyle.Width(w).Render(line)
	default:
		return dimStyle.Width(w).Render(line)
	}
}
