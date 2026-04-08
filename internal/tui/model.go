package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/chatlive"
	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// Config drives the TUI session (kernel + transcript + input).
type Config struct {
	Ctx         context.Context
	Client      core.StreamClient
	Registry    *tools.Registry
	Messages    *[]sdk.ChatCompletionMessage
	// AutoApprove toggles dangerous-tool and MCP ask-path approval (Shift+Tab in TUI). If nil, treated as off.
	AutoApprove *atomic.Bool
	Banner      string
	// MCPManager is optional; used for footer hints when servers use non-ask approval.
	MCPManager *mcpclient.Manager
	Slash       func(line string) (appendOut string, exitChat bool, err error)
	// BeforeUserTurn runs before each user-authored model turn (optional; e.g. auto-compact).
	BeforeUserTurn func() error
	// AfterTurn runs after a successful model turn (optional; e.g. persist session).
	AfterTurn func() error
	// StatusLine is shown under the title (provider · model · session).
	StatusLine string
	// ToolPreviewMax is the max UTF-8 runes of tool stdout in the transcript (0 = default 4000).
	ToolPreviewMax int
	// MarkdownAssist renders finished assistant turns with goldmark + Chroma (disable with OPENCLAUDE_TUI_MARKDOWN=0).
	MarkdownAssist bool
	// ImageURLs and ImageFiles apply to the first non-slash user message (vision); then cleared on success.
	ImageURLs  []string
	ImageFiles []string
	// Live binds the agent for /model and /provider swaps (optional).
	Live *chatlive.LiveChat
	// Busy, if non-nil, is set to 1 while a model turn runs (for slash guards).
	Busy *int32
	// StatusLineFunc, if set, overrides StatusLine each render (e.g. after /model).
	StatusLineFunc func() string
	// Theme drives lipgloss + markdown rendering (optional).
	Theme *ThemeHolder
	// VimKeys toggles vim-style prompt editing (/vim in TUI); nil disables.
	VimKeys *VimKeysHolder
	// SkillNames returns loaded skill names for /-completion (optional).
	SkillNames func() []string
}

type model struct {
	cfg               Config
	send              func(tea.Msg)
	vp                viewport.Model
	ti                textinput.Model
	busy              bool
	perm              *permState
	width             int
	height            int
	permBr            *permBridge
	getAgent          func() *core.Agent
	stickBottom       bool
	runningTool       string
	pendingImageURLs  []string
	pendingImageFiles []string
	vimNormal         bool // true = vim normal mode on prompt (movement); false = insert when VimKeys enabled
	// transcript
	committed strings.Builder
	liveAsst  strings.Builder

	slashAll             []slashEntry
	slashMatches         []slashEntry
	slashSel             int
	slashSuggestIsArg    bool // true = first-arg completion; render primary without leading /
	slashEscDismiss      bool
	slashDismissSnapshot string
	compMode             int
	replaceStart         int // byte offsets into ti.Value for arg/path/skill replace; -1 when unused
	replaceEnd           int
	inputHistory         []string
	historyIdx           int // -1 = editing; else index into inputHistory (newest at len-1)
	historyDraft         string
	historyFilterActive  bool   // prefix search mode (non-empty line + ↑)
	historyFilterMatches []int  // indices into inputHistory, newest-first
	historyFilterPos     int    // position in historyFilterMatches
	userSubmitCount      int // non-slash user messages sent to the model
	toastText            string
	toastKind            int
	toastClearID         int
	pendingToastCmd      tea.Cmd
}

type permState struct {
	tool string
	args map[string]any
	ch   chan bool
}

type kernelMsg struct {
	e core.Event
}

type runTurnDoneMsg struct {
	clearImages bool
}

type runTurnErrMsg struct {
	err error
}

func newModel(cfg Config, send func(tea.Msg), getAgent func() *core.Agent, pb *permBridge) *model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 72

	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	m := &model{
		cfg:               cfg,
		send:              send,
		vp:                vp,
		ti:                ti,
		permBr:            pb,
		getAgent:          getAgent,
		stickBottom:       true,
		pendingImageURLs:  append([]string(nil), cfg.ImageURLs...),
		pendingImageFiles: append([]string(nil), cfg.ImageFiles...),
		slashAll:          buildSlashIndex(cfg.SkillNames),
		historyIdx:        -1,
		replaceStart:      -1,
	}
	m.syncPlaceholder()
	if cfg.Banner != "" {
		if strings.ContainsRune(cfg.Banner, '\x1b') {
			m.committed.WriteString(cfg.Banner)
		} else {
			m.committed.WriteString(dimStyle.Render(cfg.Banner))
		}
		m.committed.WriteByte('\n')
	}
	m.syncVP()
	return m
}

func (m *model) toolPreviewLimit() int {
	if m.cfg.ToolPreviewMax > 0 {
		return m.cfg.ToolPreviewMax
	}
	return 4000
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.reflowLayout()
		return m, nil

	case tea.MouseMsg:
		if m.perm == nil {
			if msg.Action == tea.MouseActionPress {
				switch msg.Button { //nolint:exhaustive
				case tea.MouseButtonWheelUp:
					m.stickBottom = false
				}
			}
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown && m.vp.AtBottom() {
				m.stickBottom = true
			}
			oldInput := m.ti.Value()
			m.ti, cmd = m.ti.Update(msg)
			if m.ti.Value() != oldInput {
				m.resetHistoryNavigationOnEdit()
				if m.compMode == compFile || m.compMode == compSkill {
					m.clearSuggestOverlay()
				}
			}
			m.syncSuggestOverlay()
			m.reflowLayout()
			return m, cmd
		}

	case tea.KeyMsg:
		if m.tryQuestionMarkHelp(msg) {
			m.syncSuggestOverlay()
			m.reflowLayout()
			return m, textinput.Blink
		}
		if m.slashSuggestActive() {
			switch msg.Type { //nolint:exhaustive
			case tea.KeyTab:
				m.applySuggestCompletion()
				m.reflowLayout()
				return m, textinput.Blink
			case tea.KeyShiftTab:
				if len(m.slashMatches) > 1 {
					m.slashSel = (m.slashSel - 1 + len(m.slashMatches)) % len(m.slashMatches)
				}
				return m, nil
			case tea.KeyUp:
				if len(m.slashMatches) > 1 {
					m.slashSel = (m.slashSel - 1 + len(m.slashMatches)) % len(m.slashMatches)
				}
				return m, nil
			case tea.KeyDown:
				if len(m.slashMatches) > 1 {
					m.slashSel = (m.slashSel + 1) % len(m.slashMatches)
				}
				return m, nil
			case tea.KeyEsc:
				m.slashEscDismiss = true
				m.slashDismissSnapshot = m.ti.Value()
				m.rebuildSlashMatches()
				m.reflowLayout()
				return m, nil
			}
		}
		if m.perm != nil {
			switch strings.ToLower(msg.String()) {
			case "y":
				m.answerPerm(true)
				m.reflowLayout()
				return m, nil
			case "n":
				m.answerPerm(false)
				m.reflowLayout()
				return m, nil
			case "esc", "q":
				m.answerPerm(false)
				m.reflowLayout()
				return m, nil
			}
			return m, nil
		}
		if msg.Type == tea.KeyShiftTab && !m.busy && !m.slashSuggestActive() {
			if m.cfg.AutoApprove != nil {
				v := !m.cfg.AutoApprove.Load()
				m.cfg.AutoApprove.Store(v)
				if v {
					m.commitLine(dimStyle.Render("auto-approve: on"))
				} else {
					m.commitLine(dimStyle.Render("auto-approve: off"))
				}
			}
			return m, nil
		}
		vimNor := m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() && m.vimNormal
		if msg.Type == tea.KeyTab && !m.busy && m.perm == nil && !m.slashSuggestActive() && !vimNor {
			if m.tryExpandNonSlashTab() {
				m.syncSuggestOverlay()
				m.reflowLayout()
				return m, textinput.Blink
			}
		}
		if !m.slashSuggestActive() && m.perm == nil && !m.busy {
			vimBlockHist := m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() && m.vimNormal
			if !vimBlockHist {
				if msg.Type == tea.KeyUp {
					if ok, hcmd := m.historyUp(); ok {
						m.syncSuggestOverlay()
						m.reflowLayout()
						if hcmd != nil {
							return m, tea.Batch(textinput.Blink, hcmd)
						}
						return m, textinput.Blink
					}
				}
				if msg.Type == tea.KeyDown {
					if ok, hcmd := m.historyDown(); ok {
						m.syncSuggestOverlay()
						m.reflowLayout()
						if hcmd != nil {
							return m, tea.Batch(textinput.Blink, hcmd)
						}
						return m, textinput.Blink
					}
				}
			}
		}
		switch msg.Type {
		case tea.KeyPgUp:
			m.stickBottom = false
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		case tea.KeyPgDown:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			if m.vp.AtBottom() {
				m.stickBottom = true
			}
			return m, cmd
		case tea.KeyHome:
			m.stickBottom = false
			m.vp.GotoTop()
			return m, nil
		case tea.KeyEnd:
			m.stickBottom = true
			m.vp.GotoBottom()
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

		vimOn := m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() && !m.busy
		if !vimOn {
			if m.vimNormal {
				m.vimNormal = false
				m.reflowLayout()
			}
		}
		if vimOn && !m.vimNormal && msg.Type == tea.KeyEsc {
			m.vimNormal = true
			m.reflowLayout()
			return m, textinput.Blink
		}
		if vimOn && m.vimNormal {
			m.handleVimNormalKey(msg)
			return m, textinput.Blink
		}

	case kernelMsg:
		m.applyKernel(msg.e)
		c := m.pendingToastCmd
		m.pendingToastCmd = nil
		if c != nil {
			return m, tea.Batch(textinput.Blink, c)
		}
		return m, nil

	case toastClearMsg:
		if msg.id == m.toastClearID {
			m.toastText = ""
			m.reflowLayout()
		}
		return m, nil

	case permPromptMsg:
		m.perm = &permState{tool: msg.tool, args: msg.args, ch: msg.result}
		m.rebuildSlashMatches()
		m.reflowLayout()
		return m, nil

	case runTurnDoneMsg:
		m.runningTool = ""
		m.setBusy(false)
		if msg.clearImages {
			m.pendingImageURLs = nil
			m.pendingImageFiles = nil
		}
		if m.cfg.AfterTurn != nil {
			if err := m.cfg.AfterTurn(); err != nil {
				m.commitLine(errStyle.Render("session save: ") + err.Error())
				if c := m.pushToast(err.Error(), toastErr); c != nil {
					return m, tea.Batch(textinput.Blink, c)
				}
			}
		}
		return m, textinput.Blink

	case runTurnErrMsg:
		m.runningTool = ""
		m.setBusy(false)
		var c tea.Cmd
		if msg.err != nil {
			c = m.pushToast(msg.err.Error(), toastErr)
		}
		if c != nil {
			return m, tea.Batch(textinput.Blink, c)
		}
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	if m.perm == nil {
		oldInput := m.ti.Value()
		m.vp, cmd = m.vp.Update(msg)
		m.ti, cmd = m.ti.Update(msg)
		if m.ti.Value() != oldInput {
			m.resetHistoryNavigationOnEdit()
			if m.compMode == compFile || m.compMode == compSkill {
				m.clearSuggestOverlay()
			}
		}
		m.syncSuggestOverlay()
		m.reflowLayout()
	}
	return m, cmd
}

func (m *model) slashSuggestActive() bool {
	return len(m.slashMatches) > 0
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
	m.rebuildSlashMatches()
}

func (m *model) submitLine(line string) (tea.Model, tea.Cmd) {
	if line == "" {
		return m, textinput.Blink
	}
	m.appendHistory(line)
	m.resetHistoryNavigation()
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
			if c := m.pushToast(err.Error(), toastErr); c != nil {
				return m, tea.Batch(textinput.Blink, c)
			}
			return m, textinput.Blink
		}
		if strings.TrimSpace(out) != "" {
			m.commitLine(out)
		}
		return m, nil
	}

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

	if m.cfg.BeforeUserTurn != nil {
		if err := m.cfg.BeforeUserTurn(); err != nil {
			m.setBusy(false)
			m.commitLine(errStyle.Render("before turn: ") + err.Error())
			if c := m.pushToast(err.Error(), toastErr); c != nil {
				return m, tea.Batch(textinput.Blink, c)
			}
			return m, textinput.Blink
		}
	}

	send := m.send
	ctx := m.cfg.Ctx
	urls := append([]string(nil), m.pendingImageURLs...)
	files := append([]string(nil), m.pendingImageFiles...)
	hasVis := len(urls) > 0 || len(files) > 0
	parts, err := core.BuildUserContentParts(line, urls, files)
	if err != nil {
		m.setBusy(false)
		m.commitLine(errStyle.Render("Error: ") + err.Error())
		if c := m.pushToast(err.Error(), toastErr); c != nil {
			return m, tea.Batch(textinput.Blink, c)
		}
		return m, textinput.Blink
	}

	m.userSubmitCount++
	m.syncPlaceholder()

	go func(parts []sdk.ChatMessagePart, clear bool) {
		var err error
		if len(parts) == 1 && parts[0].Type == sdk.ChatMessagePartTypeText {
			err = ag.RunUserTurn(ctx, msgs, parts[0].Text)
		} else {
			err = ag.RunUserTurnMulti(ctx, msgs, parts)
		}
		if err != nil {
			send(runTurnErrMsg{err: err})
			return
		}
		send(runTurnDoneMsg{clearImages: clear})
	}(parts, hasVis)

	return m, nil
}

func (m *model) headerSubLines() int {
	if m.busy || m.runningTool != "" {
		return 2
	}
	return 1
}

func (m *model) applyKernel(e core.Event) {
	m.pendingToastCmd = nil
	sub0 := m.headerSubLines()
	switch e.Kind {
	case core.KindUserMessage:
		m.commitLine(userStyle.Render("You") + ": " + e.UserText)
	case core.KindAssistantTextDelta:
		m.stickBottom = true
		m.liveAsst.WriteString(e.TextChunk)
		m.syncVP()
	case core.KindAssistantFinished:
		txt := strings.TrimSpace(m.liveAsst.String())
		if txt == "" {
			txt = strings.TrimSpace(e.AssistantText)
		}
		m.liveAsst.Reset()
		if txt == "" {
			m.syncVP()
			break
		}
		m.stickBottom = true
		label := asstStyle.Render("Assistant") + ":"
		glam := "dark"
		if m.cfg.Theme != nil {
			glam = m.cfg.Theme.MarkdownStyle()
		}
		md := renderAssistantMarkdown(m.vp.Width, txt, m.cfg.MarkdownAssist, glam)
		if md != "" {
			m.committed.WriteString(lipgloss.JoinVertical(lipgloss.Left, label, md))
		} else {
			body := lipgloss.NewStyle().Width(max(40, m.vp.Width)).Render(txt)
			m.committed.WriteString(lipgloss.JoinVertical(lipgloss.Left, label, body))
		}
		m.committed.WriteByte('\n')
		m.syncVP()
	case core.KindToolCall:
		m.runningTool = e.ToolName
		args := formatToolArgs(e.ToolArgsJSON, e.ToolArgs)
		hdr := toolStyle.Render("Tool") + ": " + e.ToolName
		m.commitLine(hdr + "\n" + dimStyle.Render(args))
	case core.KindPermissionPrompt:
		// Interactive modal is driven by permPromptMsg from Confirm; no extra line.
	case core.KindPermissionResult:
		switch {
		case autoApproveEnabled(m.cfg.AutoApprove) && e.PermissionApproved:
			m.commitLine(dimStyle.Render(fmt.Sprintf("[auto-approved] %s", e.PermissionTool)))
		case e.PermissionApproved:
			m.commitLine(okStyle.Render("Approved: ") + e.PermissionTool)
		default:
			m.commitLine(errStyle.Render("Declined: ") + e.PermissionTool)
			m.pendingToastCmd = m.pushToast("Declined: "+e.PermissionTool, toastWarn)
		}
	case core.KindToolResult:
		m.runningTool = ""
		var b strings.Builder
		b.WriteString(dimStyle.Render("Result (" + e.ToolName + "): "))
		if e.ToolExecError != "" {
			b.WriteString(errStyle.Render(e.ToolExecError))
		} else {
			b.WriteString(formatToolResultBody(m.toolPreviewLimit(), e.ToolResultText, m.vp.Width))
		}
		m.commitLine(b.String())
	case core.KindError:
		m.runningTool = ""
		m.commitLine(errStyle.Render("Error: ") + e.Message)
		m.pendingToastCmd = m.pushToast(e.Message, toastErr)
	case core.KindModelRefusal:
		m.commitLine(errStyle.Render("Refused: ") + e.Message)
		m.pendingToastCmd = m.pushToast(e.Message, toastWarn)
	case core.KindTurnComplete:
		m.runningTool = ""
	}
	if m.headerSubLines() != sub0 {
		m.reflowLayout()
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

func (m *model) setBusy(v bool) {
	m.busy = v
	if m.cfg.Busy != nil {
		if v {
			atomic.StoreInt32(m.cfg.Busy, 1)
		} else {
			atomic.StoreInt32(m.cfg.Busy, 0)
		}
	}
	m.reflowLayout()
}

// reflowLayout recomputes viewport and input width from m.width/m.height and current chrome (permission panel, busy line).
func (m *model) reflowLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	subLines := 1
	if m.busy || m.runningTool != "" {
		subLines = 2
	}
	headerH := 1 + subLines
	toastH := 0
	if strings.TrimSpace(m.toastText) != "" {
		toastH = 1
	}
	suggestH := suggestionBlockHeight(m.slashMatches)
	footerH := promptChromeLines + suggestH
	if m.perm != nil {
		footerH += permPanelReserveLines
	}
	vpH := m.height - headerH - toastH - footerH
	if vpH < 6 {
		vpH = 6
	}
	vpW := m.width - 2
	if vpW < 20 {
		vpW = 20
	}
	m.vp.Width = vpW
	m.vp.Height = vpH
	m.ti.Width = m.textInputWidth()
	m.syncVP()
}

func (m *model) commitLine(s string) {
	m.stickBottom = true
	m.committed.WriteString(s)
	m.committed.WriteByte('\n')
	m.syncVP()
}

func (m *model) assistantMarkdownTheme() string {
	if m.cfg.Theme == nil {
		return "dark"
	}
	return m.cfg.Theme.MarkdownStyle()
}

func (m *model) syncVP() {
	committed := m.committed.String()
	liveRaw := m.liveAsst.String()
	live := liveRaw
	if m.cfg.MarkdownAssist && strings.TrimSpace(liveRaw) != "" {
		s := strings.ToLower(strings.TrimSpace(m.assistantMarkdownTheme()))
		dark := s != "light"
		if md := renderAssistantMarkdownChroma(m.vp.Width, liveRaw, dark, false); md != "" {
			live = md
		}
	}
	m.vp.SetContent(committed + live)
	if m.stickBottom {
		m.vp.GotoBottom()
	}
}

func (m *model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	header := titleStyle.Width(m.width).Render("OpenClaude v4 — TUI")
	status := strings.TrimSpace(m.cfg.StatusLine)
	if m.cfg.StatusLineFunc != nil {
		if s := strings.TrimSpace(m.cfg.StatusLineFunc()); s != "" {
			status = s
		}
	}
	if status == "" {
		status = "Ctrl+C quit · PgUp/PgDn Home/End · ↑↓ history · prefix+↑ · Tab paths · ? · /help"
	} else {
		status = status + "    PgUp/PgDn Home/End · ↑↓ hist · prefix+↑ · Tab · ? · /help"
	}
	sub := dimStyle.Width(m.width).Render(status)
	if m.busy || m.runningTool != "" {
		var w strings.Builder
		w.WriteString("working…")
		if m.runningTool != "" {
			w.WriteString(" · ")
			w.WriteString(m.runningTool)
		}
		sub = lipgloss.JoinVertical(lipgloss.Left, sub, dimStyle.Width(m.width).Render(w.String()))
	}
	toastLine := m.renderToastLine()
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

	var vimSeg string
	if m.cfg.VimKeys != nil && m.cfg.VimKeys.Enabled() {
		if m.vimNormal {
			vimSeg = dimStyle.Render("(vim NOR) ")
		} else {
			vimSeg = dimStyle.Render("(vim INS) ")
		}
	}
	inner := lipgloss.JoinHorizontal(lipgloss.Left, vimSeg, m.promptCharRendered(), m.ti.View())
	inputLine := promptBoxStyle.Width(m.width).Render(inner)
	rule := dimStyle.Width(m.width).Render(horizontalRule(m.width))
	th := config.SessionCompactTokenThreshold()
	left := buildFooterLeft(autoApproveEnabled(m.cfg.AutoApprove), m.cfg.MCPManager)
	right := buildCompactMeterRight(m.cfg.Messages, th)
	if th <= 0 {
		right = "auto-compact off"
	}
	hintRow := formatFooterRow(left, right, m.width)
	promptStack := lipgloss.JoinVertical(lipgloss.Left, rule, inputLine, rule, hintRow)

	slashBlock := renderSlashSuggestions(m.width, m.slashMatches, m.slashSel, m.slashSuggestIsArg)

	rows := []string{header, sub}
	if toastLine != "" {
		rows = append(rows, toastLine)
	}
	rows = append(rows, body)
	if permBlock != "" {
		rows = append(rows, permBlock)
	}
	if slashBlock != "" {
		rows = append(rows, slashBlock)
	}
	rows = append(rows, promptStack)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
