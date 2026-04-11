package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/providerwizard"
)

type providerWiz struct {
	wiz *providerwizard.Wizard
	ti  textinput.Model
}

func newProviderWiz(termW int) *providerWiz {
	if termW < 24 {
		termW = 80
	}
	tw := termW - 8
	if tw < 20 {
		tw = 20
	}
	w := providerwizard.New()
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 4096
	ti.Width = tw
	p := &providerWiz{wiz: w, ti: ti}
	p.syncTextInputFromWizard()
	return p
}

func (p *providerWiz) syncTextInputFromWizard() {
	if p.wiz.StepKind() == providerwizard.StepText {
		p.ti.SetValue(p.wiz.TextDefault())
		p.ti.CursorEnd()
	} else {
		p.ti.SetValue("")
	}
}

func (m *model) finishProviderWiz() (tea.Model, tea.Cmd) {
	if m.pwiz == nil {
		return m, textinput.Blink
	}
	w := m.pwiz.wiz
	if w.Cancelled() {
		m.commitLine(dimStyle.Render("(provider wizard cancelled)"))
	} else {
		m.commitLine(w.Result())
		m.commitLine(dimStyle.Render("Restart openclaude after saving config changes."))
	}
	m.pwiz = nil
	m.reflowLayout()
	return m, textinput.Blink
}

func (m *model) updateProviderWizKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pwiz == nil {
		return m, nil
	}
	w := m.pwiz.wiz
	switch w.StepKind() {
	case providerwizard.StepDone:
		return m.finishProviderWiz()
	case providerwizard.StepMenu:
		return m.updateProviderWizMenu(msg)
	case providerwizard.StepText:
		return m.updateProviderWizText(msg)
	default:
		return m, nil
	}
}

func (m *model) updateProviderWizMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	w := m.pwiz.wiz
	switch msg.Type { //nolint:exhaustive
	case tea.KeyUp:
		w.MenuMove(-1)
		return m, nil
	case tea.KeyDown:
		w.MenuMove(1)
		return m, nil
	case tea.KeyEnter:
		if err := w.SelectCurrentMenu(); err != nil {
			m.commitLine(errStyle.Render(err.Error()))
			return m, nil
		}
		if w.Finished() {
			return m.finishProviderWiz()
		}
		m.pwiz.syncTextInputFromWizard()
		return m, textinput.Blink
	case tea.KeyEsc:
		w.Cancel()
		return m.finishProviderWiz()
	}
	switch strings.ToLower(msg.String()) {
	case "b", "back":
		if !w.Back() && w.IsProviderMenu() {
			w.Cancel()
			return m.finishProviderWiz()
		}
		m.pwiz.syncTextInputFromWizard()
		return m, nil
	case "q":
		w.Cancel()
		return m.finishProviderWiz()
	}
	return m, nil
}

func (m *model) updateProviderWizText(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	w := m.pwiz.wiz
	switch msg.Type { //nolint:exhaustive
	case tea.KeyEsc:
		w.Cancel()
		return m.finishProviderWiz()
	case tea.KeyEnter:
		if err := w.SubmitText(m.pwiz.ti.Value()); err != nil {
			m.commitLine(errStyle.Render(err.Error()))
			return m, nil
		}
		if w.Finished() {
			return m.finishProviderWiz()
		}
		m.pwiz.syncTextInputFromWizard()
		return m, textinput.Blink
	}
	if s := strings.ToLower(strings.TrimSpace(msg.String())); s == "b" || s == "back" {
		_ = w.Back()
		if w.Finished() {
			return m.finishProviderWiz()
		}
		m.pwiz.syncTextInputFromWizard()
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.pwiz.ti, cmd = m.pwiz.ti.Update(msg)
	return m, cmd
}

func providerWizMenuHint(w *providerwizard.Wizard, opt string) string {
	if w.IsProviderMenu() {
		switch strings.ToLower(strings.TrimSpace(opt)) {
		case "openai":
			return "OpenAI or compatible API"
		case "ollama":
			return "Local Ollama"
		case "gemini":
			return "Google Gemini"
		case "github":
			return "GitHub Models"
		case "openrouter":
			return "OpenRouter"
		}
	}
	if w.IsOllamaModelMenu() && strings.HasPrefix(opt, "Other ") {
		return "Type model name manually"
	}
	return ""
}

// renderProviderWizSlashMenu draws the wizard menu like slash-command suggestions
// (dim header, bordered list, purple highlight on the selected row).
func renderProviderWizSlashMenu(contentW int, opts []string, cursor int, w *providerwizard.Wizard) string {
	if len(opts) == 0 || contentW < 1 {
		return ""
	}
	entries := make([]slashEntry, 0, len(opts))
	for _, opt := range opts {
		entries = append(entries, slashEntry{primary: opt, hint: providerWizMenuHint(w, opt)})
	}
	argMode := true
	innerW := max(1, contentW-4)
	header := dimStyle.Width(contentW).Render("↑↓ select · Enter confirm · b back · Esc cancel")
	col1W, colGap := slashSuggestColumnWidths(innerW, entries, argMode)
	menuRows := make([]string, 0, len(entries))
	for i, e := range entries {
		menuRows = append(menuRows, renderSlashSuggestRow(innerW, col1W, colGap, e, argMode, i == cursor))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, menuRows...)
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(max(3, contentW-2)).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, box)
}

func (m *model) renderProviderWizPanel() string {
	if m.pwiz == nil {
		return ""
	}
	w := m.pwiz.wiz
	innerW := m.width - 4
	if innerW < 20 {
		innerW = 20
	}
	var rows []string
	rows = append(rows, titleStyle.Render(w.Title()))
	if b := w.Body(); strings.TrimSpace(b) != "" {
		rows = append(rows, dimStyle.Width(innerW).Render(b))
	}
	switch w.StepKind() {
	case providerwizard.StepMenu:
		rows = append(rows, renderProviderWizSlashMenu(innerW, w.MenuOptions(), w.MenuCursor(), w))
	case providerwizard.StepText:
		if h := w.TextHint(); strings.TrimSpace(h) != "" {
			rows = append(rows, dimStyle.Width(innerW).Render(h))
		}
		rows = append(rows, fmt.Sprintf("%s: %s", w.TextLabel(), m.pwiz.ti.View()))
	}
	if w.StepKind() != providerwizard.StepMenu {
		rows = append(rows, dimStyle.Width(innerW).Render(w.HintLine()))
	}
	box := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.NewStyle().
		Width(m.width - 2).
		Border(lipgloss.NormalBorder()).
		Padding(1, 2).
		Render(box)
}
