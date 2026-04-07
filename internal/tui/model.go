package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// Config drives the TUI session (kernel + transcript + input).
type Config struct {
	Ctx         context.Context
	Client      core.StreamClient
	Registry    *tools.Registry
	Messages    *[]sdk.ChatCompletionMessage
	AutoApprove bool
	Banner      string
	Slash       func(line string) (appendOut string, exitChat bool, err error)
	// BeforeUserTurn runs before each user-authored model turn (optional; e.g. auto-compact).
	BeforeUserTurn func() error
	// AfterTurn runs after a successful model turn (optional; e.g. persist session).
	AfterTurn func() error
}

type model struct {
	cfg      Config
	send     func(tea.Msg)
	vp       viewport.Model
	ti       textinput.Model
	busy     bool
	perm     *permState
	width    int
	height   int
	permBr   *permBridge
	getAgent func() *core.Agent
	// transcript
	committed strings.Builder
	liveAsst  strings.Builder
}

type permState struct {
	tool string
	args map[string]any
	ch   chan bool
}

type kernelMsg struct {
	e core.Event
}

type runTurnDoneMsg struct{}

type runTurnErrMsg struct {
	err error
}

func newModel(cfg Config, send func(tea.Msg), getAgent func() *core.Agent, pb *permBridge) *model {
	ti := textinput.New()
	ti.Placeholder = "Message… (Enter to send, /help, Ctrl+C quit)"
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 72

	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	m := &model{
		cfg:      cfg,
		send:     send,
		vp:       vp,
		ti:       ti,
		permBr:   pb,
		getAgent: getAgent,
	}
	if cfg.Banner != "" {
		m.committed.WriteString(dimStyle.Render(cfg.Banner))
		m.committed.WriteByte('\n')
	}
	m.syncVP()
	return m
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 2
		footerH := 4
		vpH := msg.Height - headerH - footerH
		if vpH < 6 {
			vpH = 6
		}
		vpW := msg.Width - 2
		if vpW < 20 {
			vpW = 20
		}
		m.vp.Width = vpW
		m.vp.Height = vpH
		m.ti.Width = vpW - 2
		m.syncVP()
		return m, nil

	case tea.KeyMsg:
		if m.perm != nil {
			switch strings.ToLower(msg.String()) {
			case "y":
				m.answerPerm(true)
				return m, nil
			case "n":
				m.answerPerm(false)
				return m, nil
			case "esc", "q":
				m.answerPerm(false)
				return m, nil
			}
			return m, nil
		}
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if msg.Type == tea.KeyEnter && !m.busy {
			line := strings.TrimSpace(m.ti.Value())
			m.ti.SetValue("")
			return m.submitLine(line)
		}

	case kernelMsg:
		m.applyKernel(msg.e)
		return m, nil

	case permPromptMsg:
		m.perm = &permState{tool: msg.tool, args: msg.args, ch: msg.result}
		return m, nil

	case runTurnDoneMsg:
		m.busy = false
		if m.cfg.AfterTurn != nil {
			if err := m.cfg.AfterTurn(); err != nil {
				m.commitLine(errStyle.Render("session save: ") + err.Error())
			}
		}
		return m, textinput.Blink

	case runTurnErrMsg:
		m.busy = false
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	if m.perm == nil {
		m.vp, cmd = m.vp.Update(msg)
		m.ti, cmd = m.ti.Update(msg)
	}
	return m, cmd
}

func (m *model) answerPerm(ok bool) {
	if m.perm == nil {
		return
	}
	ch := m.perm.ch
	m.perm = nil
	select {
	case ch <- ok:
	default:
	}
}

func (m *model) submitLine(line string) (tea.Model, tea.Cmd) {
	if line == "" {
		return m, textinput.Blink
	}
	if strings.HasPrefix(line, "/") {
		if m.cfg.Slash == nil {
			m.commitLine(errStyle.Render("slash handler not configured"))
			return m, nil
		}
		out, exit, err := m.cfg.Slash(line)
		if exit {
			return m, tea.Quit
		}
		if err != nil {
			m.commitLine(errStyle.Render("Error: ") + err.Error())
			return m, nil
		}
		if strings.TrimSpace(out) != "" {
			m.commitLine(out)
		}
		return m, nil
	}

	m.busy = true
	ag := m.getAgent()
	if ag == nil {
		m.busy = false
		m.commitLine(errStyle.Render("Error: agent not ready"))
		return m, nil
	}
	msgs := m.cfg.Messages
	if msgs == nil {
		m.busy = false
		m.commitLine(errStyle.Render("Error: messages buffer nil"))
		return m, nil
	}

	if m.cfg.BeforeUserTurn != nil {
		if err := m.cfg.BeforeUserTurn(); err != nil {
			m.busy = false
			m.commitLine(errStyle.Render("before turn: ") + err.Error())
			return m, textinput.Blink
		}
	}

	send := m.send
	ctx := m.cfg.Ctx
	go func(user string) {
		err := ag.RunUserTurn(ctx, msgs, user)
		if err != nil {
			send(runTurnErrMsg{err: err})
			return
		}
		send(runTurnDoneMsg{})
	}(line)

	return m, nil
}

func (m *model) applyKernel(e core.Event) {
	switch e.Kind {
	case core.KindUserMessage:
		m.commitLine(userStyle.Render("You") + ": " + e.UserText)
	case core.KindAssistantTextDelta:
		m.liveAsst.WriteString(e.TextChunk)
		m.syncVP()
	case core.KindAssistantFinished:
		if m.liveAsst.Len() > 0 {
			txt := m.liveAsst.String()
			m.liveAsst.Reset()
			wrapped := lipgloss.NewStyle().Width(max(40, m.vp.Width)).Render(asstStyle.Render("Assistant") + ": " + txt)
			m.committed.WriteString(wrapped)
			if !strings.HasSuffix(wrapped, "\n") {
				m.committed.WriteByte('\n')
			}
			m.syncVP()
		}
	case core.KindToolCall:
		args := formatToolArgs(e.ToolArgsJSON, e.ToolArgs)
		hdr := toolStyle.Render("Tool") + ": " + e.ToolName
		m.commitLine(hdr + "\n" + dimStyle.Render(args))
	case core.KindPermissionPrompt:
		// Interactive modal is driven by permPromptMsg from Confirm; no extra line.
	case core.KindPermissionResult:
		switch {
		case m.cfg.AutoApprove && e.PermissionApproved:
			m.commitLine(dimStyle.Render(fmt.Sprintf("[auto-approved] %s", e.PermissionTool)))
		case e.PermissionApproved:
			m.commitLine(okStyle.Render("Approved: ") + e.PermissionTool)
		default:
			m.commitLine(errStyle.Render("Declined: ") + e.PermissionTool)
		}
	case core.KindToolResult:
		var b strings.Builder
		b.WriteString(dimStyle.Render("Result (" + e.ToolName + "): "))
		if e.ToolExecError != "" {
			b.WriteString(errStyle.Render(e.ToolExecError))
		} else {
			out := strings.TrimSpace(e.ToolResultText)
			if len(out) > 400 {
				out = out[:397] + "..."
			}
			b.WriteString(out)
		}
		m.commitLine(b.String())
	case core.KindError:
		m.commitLine(errStyle.Render("Error: ") + e.Message)
	case core.KindModelRefusal:
		m.commitLine(errStyle.Render("Refused: ") + e.Message)
	case core.KindTurnComplete:
		// end of turn; optional visual break
	}
}

func formatToolArgs(rawJSON string, m map[string]any) string {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON != "" && len(rawJSON) <= 400 {
		return rawJSON
	}
	if rawJSON != "" {
		return rawJSON[:397] + "..."
	}
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprint(m)
	}
	s := string(b)
	if len(s) > 400 {
		return s[:397] + "..."
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *model) commitLine(s string) {
	m.committed.WriteString(s)
	m.committed.WriteByte('\n')
	m.syncVP()
}

func (m *model) syncVP() {
	m.vp.SetContent(m.committed.String() + m.liveAsst.String())
	m.vp.GotoBottom()
}

func (m *model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	header := titleStyle.Width(m.width).Render("OpenClaude v4 — TUI")
	sub := dimStyle.Width(m.width).Render("Ctrl+C quit · /help")
	if m.busy {
		sub = lipgloss.JoinHorizontal(lipgloss.Left, sub, "  ", dimStyle.Render("working…"))
	}
	body := border.Width(m.width - 2).Render(m.vp.View())

	var permBlock string
	if m.perm != nil {
		args := formatToolArgs("", m.perm.args)
		box := lipgloss.JoinVertical(
			lipgloss.Left,
			errStyle.Bold(true).Render("Permission required"),
			"",
			fmt.Sprintf("Tool: %s", m.perm.tool),
			dimStyle.Render(args),
			"",
			okStyle.Render("[y]")+" approve  "+dimStyle.Render("[n]")+" deny  "+dimStyle.Render("[esc]")+" deny",
		)
		permBlock = lipgloss.NewStyle().
			Width(m.width-2).
			Border(lipgloss.DoubleBorder()).
			Padding(1, 2).
			Render(box)
	}

	inputLabel := "> "
	inputLine := inputLabel + m.ti.View()
	footer := lipgloss.NewStyle().Width(m.width).Render(inputLine)

	rows := []string{header, sub, body}
	if permBlock != "" {
		rows = append(rows, permBlock)
	}
	rows = append(rows, footer)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
