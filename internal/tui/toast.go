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

// Max queued in-app toasts (Crush-style backlog; each shows ~5s in order).
const toastQueueMax = 24

type toastClearMsg struct {
	id int
}

type queuedToast struct {
	text string
	kind int
}

func (m *model) toastVisible() bool {
	return len(m.toastQueue) > 0
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
	avail := w - 2
	if avail < 4 {
		avail = w - 1
		if avail < 1 {
			avail = 1
		}
	}
	if lipgloss.Width(text) > avail {
		text = truncateVisual(text, avail)
	}
	if len(m.toastQueue) >= toastQueueMax {
		return nil
	}
	m.toastQueue = append(m.toastQueue, queuedToast{text: text, kind: kind})
	if len(m.toastQueue) == 1 {
		return m.scheduleToastHead()
	}
	return nil
}

func (m *model) scheduleToastHead() tea.Cmd {
	if len(m.toastQueue) == 0 {
		return nil
	}
	m.toastClearID++
	id := m.toastClearID
	m.reflowLayout()
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{id: id}
	})
}

func (m *model) handleToastClear(msg toastClearMsg) tea.Cmd {
	if msg.id != m.toastClearID {
		return nil
	}
	if len(m.toastQueue) == 0 {
		return nil
	}
	m.toastQueue = m.toastQueue[1:]
	if len(m.toastQueue) == 0 {
		m.reflowLayout()
		return nil
	}
	return m.scheduleToastHead()
}

func (m *model) renderToastLine() string {
	if len(m.toastQueue) == 0 {
		return ""
	}
	head := m.toastQueue[0]
	w := m.width
	if w < 1 {
		w = 80
	}
	pad := 2
	avail := w - pad
	if avail < 4 {
		avail = w - 1
		if avail < 1 {
			avail = 1
		}
		pad = w - avail
	}
	text := head.text
	if lipgloss.Width(text) > avail {
		text = truncateVisual(text, avail)
	}
	prefix := strings.Repeat(" ", pad)
	plain := prefix + text
	switch head.kind {
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
