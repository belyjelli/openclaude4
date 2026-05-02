package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// toolPickPanelReserveLines is the footer height reserved while the iteration-limit tool picker is open.
const toolPickPanelReserveLines = 22

// toolPickMsg requests the TUI to open the tool subset dialog after an iteration limit error.
type toolPickMsg struct {
	parts         []sdk.ChatMessagePart
	slashAllow    []string
	hasVis        bool
	maxIterations int
	candidates    []string
}

type toolPickState struct {
	names         []string
	checked       map[string]bool
	sel           int
	scroll        int
	parts         []sdk.ChatMessagePart
	hasVis        bool
	maxIterations int
	maxToolsInAPI int
}

func effectiveToolPickCap(maxIter int) int {
	if maxIter <= 0 {
		return core.DefaultMaxIterations
	}
	return maxIter
}

const toolPickVisibleRows = 10

func toolPickCandidates(reg *tools.Registry, slashAllow []string) []string {
	return tools.IterLimitPickCandidates(reg, slashAllow)
}

func newToolPickState(parts []sdk.ChatMessagePart, hasVis bool, names []string, maxIterations int) *toolPickState {
	checked := make(map[string]bool, len(names))
	for _, n := range names {
		checked[n] = true
	}
	cpParts := make([]sdk.ChatMessagePart, len(parts))
	copy(cpParts, parts)
	capTools := effectiveToolPickCap(maxIterations)
	return &toolPickState{
		names:         names,
		checked:       checked,
		parts:         cpParts,
		hasVis:        hasVis,
		maxIterations: maxIterations,
		maxToolsInAPI: capTools,
	}
}

func (s *toolPickState) enabledCount() int {
	n := 0
	for _, name := range s.names {
		if s.checked[name] {
			n++
		}
	}
	return n
}

func (s *toolPickState) enabledNames() []string {
	var out []string
	for _, name := range s.names {
		if s.checked[name] {
			out = append(out, name)
		}
	}
	return out
}

func (s *toolPickState) validSelection(maxTools int) bool {
	if maxTools <= 0 {
		maxTools = core.DefaultMaxIterations
	}
	c := s.enabledCount()
	return c >= 1 && c <= maxTools
}

func (s *toolPickState) toggleAtSel() {
	if s == nil || s.sel < 0 || s.sel >= len(s.names) {
		return
	}
	name := s.names[s.sel]
	on := s.checked[name]
	if on {
		if s.enabledCount() <= 1 {
			return
		}
		s.checked[name] = false
		return
	}
	cap := s.maxToolsInAPI
	if cap <= 0 {
		cap = core.DefaultMaxIterations
	}
	if s.enabledCount() >= cap {
		return
	}
	s.checked[name] = true
}

func (s *toolPickState) move(delta int) {
	if s == nil || len(s.names) == 0 {
		return
	}
	s.sel = (s.sel + delta + len(s.names)) % len(s.names)
	s.ensureScroll()
}

func (s *toolPickState) ensureScroll() {
	if s.sel < s.scroll {
		s.scroll = s.sel
	}
	if s.sel >= s.scroll+toolPickVisibleRows {
		s.scroll = s.sel - toolPickVisibleRows + 1
	}
}

func (m *model) renderToolPickPanel(innerW int) string {
	s := m.toolPick
	if s == nil {
		return ""
	}
	maxTools := s.maxToolsInAPI
	if maxTools <= 0 {
		maxTools = core.DefaultMaxIterations
	}
	maxRounds := s.maxIterations
	if maxRounds <= 0 {
		maxRounds = core.DefaultMaxIterations
	}
	nEn := s.enabledCount()
	hint := fmt.Sprintf("Uncheck tools until at most %d are enabled (currently %d). Space toggles · ↑↓ · Enter retry · Esc cancel", maxTools, nEn)
	if nEn > maxTools {
		hint = errStyle.Width(innerW).Render(hint)
	} else if nEn == 0 {
		hint = errStyle.Width(innerW).Render("Enable at least one tool. Space toggles · ↑↓ · Enter retry · Esc cancel")
	} else {
		hint = dimStyle.Width(innerW).Render(hint)
	}

	end := s.scroll + toolPickVisibleRows
	if end > len(s.names) {
		end = len(s.names)
	}
	var rows []string
	for i := s.scroll; i < end; i++ {
		name := s.names[i]
		on := s.checked[name]
		box := "[ ]"
		if on {
			box = "[x]"
		}
		line := fmt.Sprintf("%s  %s", box, name)
		if i == s.sel {
			line = selectionStyle.Render(line)
		} else {
			line = dimStyle.Render(line)
		}
		rows = append(rows, line)
	}
	if len(s.names) > toolPickVisibleRows {
		rows = append(rows, dimStyle.Width(innerW).Render(fmt.Sprintf("— rows %d–%d of %d —", s.scroll+1, end, len(s.names))))
	}
	list := lipgloss.JoinVertical(lipgloss.Left, rows...)

	box := lipgloss.JoinVertical(
		lipgloss.Left,
		errStyle.Bold(true).Render("Tool iteration limit"),
		"",
		dimStyle.Width(innerW).Render(fmt.Sprintf("The model hit the %d-round tool cap. Trim which tools are sent to the API (≤%d). Task is omitted in scoped retry.", maxRounds, maxTools)),
		"",
		list,
		"",
		hint,
	)
	return lipgloss.NewStyle().
		Width(m.width-2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Render(box)
}

// selectionStyle highlights the current tool row (reuse perm-adjacent contrast).
var selectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)

func (m *model) handleToolPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.toolPick
	if s == nil {
		return m, nil
	}
	switch msg.Type { //nolint:exhaustive
	case tea.KeyUp:
		s.move(-1)
		return m, nil
	case tea.KeyDown:
		s.move(1)
		return m, nil
	case tea.KeySpace:
		s.toggleAtSel()
		return m, nil
	case tea.KeyEsc:
		m.toolPick = nil
		m.reflowLayout()
		if c := m.pushToast("Scoped retry cancelled", toastWarn); c != nil {
			return m, tea.Batch(textinput.Blink, c)
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		if !s.validSelection(s.maxToolsInAPI) {
			return m, nil
		}
		allow := s.enabledNames()
		parts := s.parts
		hasVis := s.hasVis
		m.toolPick = nil
		m.reflowLayout()
		return m.startScopedRetryTurn(parts, hasVis, allow)
	}
	switch strings.ToLower(msg.String()) {
	case "q":
		m.toolPick = nil
		m.reflowLayout()
		if c := m.pushToast("Scoped retry cancelled", toastWarn); c != nil {
			return m, tea.Batch(textinput.Blink, c)
		}
		return m, textinput.Blink
	}
	return m, nil
}

func (m *model) startScopedRetryTurn(parts []sdk.ChatMessagePart, hasVis bool, allow []string) (tea.Model, tea.Cmd) {
	m.stickBottom = true
	m.setBusy(true)
	ag := m.getAgent()
	if ag == nil {
		m.setBusy(false)
		m.commitLine(errStyle.Render("Error: agent not ready"))
		return m, nil
	}
	msgs := m.cfg.Messages
	if msgs == nil {
		m.setBusy(false)
		m.commitLine(errStyle.Render("Error: messages buffer nil"))
		return m, nil
	}

	send := m.send
	ctx := m.cfg.Ctx
	go func(parts []sdk.ChatMessagePart, clear bool, allow []string) {
		ag.SuppressNextUserMessageEvent = true
		var err error
		if len(parts) == 1 && parts[0].Type == sdk.ChatMessagePartTypeText {
			err = ag.RunUserTurnScoped(ctx, msgs, parts[0].Text, allow)
		} else {
			err = ag.RunUserTurnMultiScoped(ctx, msgs, parts, allow)
		}
		if err != nil {
			var ile *core.IterationLimitError
			if errors.As(err, &ile) {
				cands := toolPickCandidates(m.cfg.Registry, allow)
				if len(cands) > 0 {
					send(toolPickMsg{
						parts:         parts,
						slashAllow:    allow,
						hasVis:        clear,
						maxIterations: ile.MaxIterations,
						candidates:    cands,
					})
					return
				}
			}
			send(runTurnErrMsg{err: err})
			return
		}
		send(runTurnDoneMsg{clearImages: clear})
	}(parts, hasVis, append([]string(nil), allow...))

	return m, nextBusyTick()
}
