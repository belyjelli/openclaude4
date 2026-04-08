package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gitlawb/openclaude4/internal/chatlive"
	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/skills"
	"github.com/gitlawb/openclaude4/internal/startupbanner"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/gitlawb/openclaude4/internal/tui"
	sdk "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runChat(cmd *cobra.Command, _ []string) error {
	listSessions, _ := cmd.Flags().GetBool("list-sessions")
	if listSessions {
		dir := config.EffectiveSessionDir()
		entries, err := session.List(dir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "(no saved sessions in %s)\n", dir)
			return nil
		}
		_, _ = fmt.Fprintf(os.Stdout, "Sessions in %s:\n", dir)
		for _, e := range entries {
			ts := "(unknown)"
			if !e.Updated.IsZero() {
				ts = e.Updated.UTC().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(os.Stdout, "  %-24s  %4d msgs  %s  cwd=%s\n", e.Name, e.NMsgs, ts, e.CWD)
		}
		return nil
	}

	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	printMode := cmd.Flags().Changed("print")
	useTUIEarly, _ := cmd.Flags().GetBool("tui")
	if printMode && (useTUIEarly || envTruthy("OPENCLAUDE_TUI")) {
		return fmt.Errorf("openclaude: --print (-p) cannot be used with --tui or OPENCLAUDE_TUI=1")
	}

	client, err := providers.NewStreamClient()
	if err != nil {
		switch {
		case errors.Is(err, openaicomp.ErrMissingAPIKey):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set OPENAI_API_KEY (or use --provider ollama / gemini as appropriate).")
			return err
		case errors.Is(err, openaicomp.ErrMissingGeminiKey):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set GEMINI_API_KEY or GOOGLE_API_KEY for provider gemini.")
			return err
		case errors.Is(err, openaicomp.ErrMissingGitHubToken):
			_, _ = fmt.Fprintln(os.Stderr, "Error: set GITHUB_TOKEN or GITHUB_PAT for provider github.")
			return err
		case errors.Is(err, providers.ErrCodexNotImplemented):
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return err
	}

	live := chatlive.New(client)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	wd, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	ctx = tools.WithWorkDir(ctx, wd)

	skillCat, err := skills.Load(config.SkillDirs())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "openclaude: skills: %v (continuing without skills)\n", err)
		skillCat = skills.EmptyCatalog()
	}
	reg := tools.NewDefaultRegistry(skillCat)
	mcpMgr := mcpclient.ConnectAndRegister(ctx, reg, config.MCPServers(), os.Stderr)
	defer mcpMgr.Close()

	var agent *core.Agent
	reg.Register(core.NewTaskTool(func() *core.Agent { return agent }))

	var messages []sdk.ChatCompletionMessage
	persist, err := resolveChatPersistence(cmd, wd, &messages)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	defer func() {
		if persist != nil {
			_ = persist.Save()
		}
	}()
	if persist != nil && !printMode {
		_, _ = fmt.Fprintf(os.Stderr, "Session %q — %d message(s) in memory · %s\n",
			persist.store.ID, len(messages), persist.store.SessionPath())
	}

	beforeUserTurn := func() error {
		return session.ApplyTokenThreshold(ctx, live.Client(), &messages,
			config.SessionCompactTokenThreshold(),
			config.SessionSummarizeOverThreshold(),
			config.SessionCompactKeepMessages(),
			core.DefaultSystemPrompt,
		)
	}

	autoApprove := strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "1") ||
		strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "true")

	useTUI := useTUIEarly || envTruthy("OPENCLAUDE_TUI")

	if !printMode {
		meta := session.RunningMeta{
			PID:      os.Getpid(),
			CWD:      wd,
			TUI:      useTUI,
			Provider: config.ProviderName(),
			Model:    config.Model(),
		}
		if persist != nil {
			meta.SessionID = persist.store.ID
		}
		if cleanup, err := session.RegisterRunning(config.EffectiveSessionDir(), meta); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: running registry: %v\n", err)
		} else if cleanup != nil {
			defer cleanup()
		}
	}

	imgURLs, _ := cmd.Flags().GetStringSlice("image-url")
	imgFiles, _ := cmd.Flags().GetStringSlice("image-file")
	pendingImgURLs := imgURLs
	pendingImgFiles := imgFiles

	if printMode {
		if err := runPrintTurn(ctx, cmd, client, reg, &messages, beforeUserTurn, autoApprove, &agent, pendingImgURLs, pendingImgFiles); err != nil {
			return err
		}
		return nil
	}

	if useTUI {
		mcpLine := mcpSummaryLine(mcpMgr)
		ansi := startupbanner.UseANSISplashFor(os.Stderr)
		bannerStr := startupbanner.BannerContent(client, version, mcpLine, ansi) +
			"\n\nTUI: Ctrl+C to quit · same /commands as plain REPL."
		var busyFlag int32
		themeHolder := tui.NewThemeHolder()
		statusFn := func() string {
			return buildTUIStatusLine(live.Client(), persist)
		}
		return tui.Run(tui.Config{
			Ctx:            ctx,
			Client:         client,
			Registry:       reg,
			Messages:       &messages,
			AutoApprove:    autoApprove,
			Banner:         bannerStr,
			StatusLine:     buildTUIStatusLine(client, persist),
			StatusLineFunc: statusFn,
			Live:           live,
			Busy:           &busyFlag,
			Theme:          themeHolder,
			ToolPreviewMax: tuiToolPreviewMax(),
			MarkdownAssist: tuiMarkdownEnabled(),
			ImageURLs:      pendingImgURLs,
			ImageFiles:     pendingImgFiles,
			BeforeUserTurn: beforeUserTurn,
			AfterTurn: func() error {
				if persist != nil {
					return persist.Save()
				}
				return nil
			},
			Slash: func(line string) (string, bool, error) {
				var out bytes.Buffer
				err := handleSlashLine(line, chatState{
					messages:                &messages,
					mcpMgr:                  mcpMgr,
					client:                  client,
					live:                    live,
					persist:                 persist,
					providerWizardIn:        nil,
					allowConfigEditorWizard: true,
					skillCat:                skillCat,
					ctx:                     ctx,
					isBusy:                  func() bool { return atomic.LoadInt32(&busyFlag) != 0 },
					themeHolder:             themeHolder,
				}, &out)
				if errors.Is(err, errSlashExitChat) {
					return out.String(), true, nil
				}
				return out.String(), false, err
			},
		})
	}

	mcpLine := mcpSummaryLine(mcpMgr)
	_ = startupbanner.Write(os.Stderr, client, version, mcpLine)

	reader := bufio.NewReader(os.Stdin)

	agent = &core.Agent{
		Client:   live.Client(),
		Registry: reg,
		Out:      os.Stdout,
		Confirm: func(toolName string, args map[string]any) bool {
			if autoApprove {
				_, _ = fmt.Fprintf(os.Stderr, "[auto-approved] %s\n", toolName)
				return true
			}
			_, _ = fmt.Fprintf(os.Stderr, "Approve tool %q with args %s? [y/N]: ", toolName, core.FormatToolArgsForLog(args))
			line, err := reader.ReadString('\n')
			if err != nil {
				return false
			}
			line = strings.TrimSpace(strings.ToLower(line))
			return line == "y" || line == "yes"
		},
	}
	live.BindAgent(agent)

	pendingURLs := pendingImgURLs
	pendingFiles := pendingImgFiles

	for {
		_, _ = fmt.Fprint(os.Stdout, "> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				_, _ = fmt.Fprintln(os.Stdout)
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			err := handleSlashLine(line, chatState{
				messages:                &messages,
				mcpMgr:                  mcpMgr,
				client:                  client,
				live:                    live,
				persist:                 persist,
				providerWizardIn:        os.Stdin,
				allowConfigEditorWizard: false,
				skillCat:                skillCat,
				ctx:                     ctx,
				isBusy:                  nil,
				themeHolder:             nil,
			}, os.Stdout)
			if errors.Is(err, errSlashExitChat) {
				return nil
			}
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		}

		if err := beforeUserTurn(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		urls := append([]string(nil), pendingURLs...)
		files := append([]string(nil), pendingFiles...)
		hasVis := len(urls) > 0 || len(files) > 0
		parts, err := core.BuildUserContentParts(line, urls, files)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		if len(parts) == 1 && parts[0].Type == sdk.ChatMessagePartTypeText {
			err = agent.RunUserTurn(ctx, &messages, parts[0].Text)
		} else {
			err = agent.RunUserTurnMulti(ctx, &messages, parts)
		}
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		if hasVis {
			pendingURLs = nil
			pendingFiles = nil
		}
		if persist != nil {
			if err := persist.Save(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "session save: %v\n", err)
			}
		}
	}
}

func runPrintTurn(
	ctx context.Context,
	cmd *cobra.Command,
	client core.StreamClient,
	reg *tools.Registry,
	messages *[]sdk.ChatCompletionMessage,
	beforeUserTurn func() error,
	autoApprove bool,
	agent **core.Agent,
	imageURLs []string,
	imageFiles []string,
) error {
	printArg, _ := cmd.Flags().GetString("print")
	var prompt string
	if strings.TrimSpace(printArg) == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin for -p -: %w", err)
		}
		prompt = strings.TrimSpace(string(b))
	} else {
		prompt = strings.TrimSpace(printArg)
	}
	if prompt == "" {
		return fmt.Errorf(`print mode needs a non-empty prompt (example: -p "question" or pipe into -p -)`)
	}

	if err := beforeUserTurn(); err != nil {
		return err
	}

	parts, err := core.BuildUserContentParts(prompt, imageURLs, imageFiles)
	if err != nil {
		return err
	}

	*agent = &core.Agent{
		Client:   client,
		Registry: reg,
		Out:      io.Discard,
		Confirm: func(toolName string, args map[string]any) bool {
			if autoApprove {
				_, _ = fmt.Fprintf(os.Stderr, "[auto-approved] %s\n", toolName)
				return true
			}
			_, _ = fmt.Fprintf(os.Stderr, "[print mode] skipping tool %q (set OPENCLAUDE_AUTO_APPROVE_TOOLS=1 to allow)\n", toolName)
			return false
		},
	}
	if len(parts) == 1 && parts[0].Type == sdk.ChatMessagePartTypeText {
		err = (*agent).RunUserTurn(ctx, messages, parts[0].Text)
	} else {
		err = (*agent).RunUserTurnMulti(ctx, messages, parts)
	}
	if err != nil {
		return err
	}
	reply := lastAssistantReply(*messages)
	if reply != "" {
		_, _ = fmt.Fprintln(os.Stdout, strings.TrimRight(reply, "\r\n"))
	}
	return nil
}

func lastAssistantReply(msgs []sdk.ChatCompletionMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == sdk.ChatMessageRoleAssistant {
			return msgs[i].Content
		}
	}
	return ""
}

// chatPersist binds a [session.Store] to the in-memory transcript.
type chatPersist struct {
	dir      string
	wd       string
	store    *session.Store
	messages *[]sdk.ChatCompletionMessage
}

func (p *chatPersist) Save() error {
	if p == nil || p.store == nil || p.messages == nil {
		return nil
	}
	return p.store.Save(session.RepairTranscript(*p.messages), p.wd)
}

func (p *chatPersist) SwitchTo(id string) error {
	if p == nil {
		return fmt.Errorf("sessions not enabled")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("empty session id")
	}
	if p.store != nil {
		if err := p.Save(); err != nil {
			return fmt.Errorf("save before switch: %w", err)
		}
	}
	p.store = &session.Store{Dir: p.dir, ID: id}
	data, err := p.store.Load()
	if err != nil {
		if os.IsNotExist(err) {
			*p.messages = nil
			return nil
		}
		return err
	}
	*p.messages = session.RepairTranscript(data.Messages)
	return nil
}

func (p *chatPersist) BranchTo(id string) error {
	if p == nil {
		return fmt.Errorf("sessions not enabled")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("empty session id")
	}
	if p.store != nil {
		if err := p.Save(); err != nil {
			return fmt.Errorf("save before switch: %w", err)
		}
	}
	p.store = &session.Store{Dir: p.dir, ID: id}
	*p.messages = nil
	return p.Save()
}

func resolveChatPersistence(cmd *cobra.Command, wd string, messages *[]sdk.ChatCompletionMessage) (*chatPersist, error) {
	if config.SessionDisabled() {
		resume, _ := cmd.Flags().GetBool("resume")
		if resume || envTruthy("OPENCLAUDE_RESUME") {
			return nil, fmt.Errorf("sessions are disabled (--no-session) but resume was requested")
		}
		sFlag, _ := cmd.Flags().GetString("session")
		if strings.TrimSpace(sFlag) != "" || strings.TrimSpace(viper.GetString("session.name")) != "" {
			return nil, fmt.Errorf("sessions are disabled (--no-session) but a session name was set")
		}
		return nil, nil
	}

	resume, _ := cmd.Flags().GetBool("resume")
	if viper.GetBool("session.resume_last") {
		resume = true
	}
	if envTruthy("OPENCLAUDE_RESUME") {
		resume = true
	}
	sFlag, _ := cmd.Flags().GetString("session")
	name := strings.TrimSpace(sFlag)
	if name == "" {
		name = strings.TrimSpace(viper.GetString("session.name"))
	}
	if resume && name != "" {
		return nil, fmt.Errorf("use either --resume (or OPENCLAUDE_RESUME) or --session / OPENCLAUDE_SESSION, not both")
	}
	dir := config.EffectiveSessionDir()

	var sessID string
	switch {
	case resume:
		id, err := session.ResolveResumeID(dir)
		if err != nil || strings.TrimSpace(id) == "" {
			sessID = session.NewRandomID()
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: no last session on disk; starting new session %q\n", sessID)
		} else {
			sessID = id
		}
	case name != "":
		sessID = name
	default:
		sessID = session.NewRandomID()
	}

	st := &session.Store{Dir: dir, ID: sessID}
	p := &chatPersist{dir: dir, wd: wd, store: st, messages: messages}
	data, err := st.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		_, _ = fmt.Fprintf(os.Stderr, "openclaude: could not load session file (%v); starting with an empty transcript.\n", err)
		*messages = nil
		return p, nil
	}
	*messages = session.RepairTranscript(data.Messages)
	return p, nil
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func buildTUIStatusLine(client core.StreamClient, persist *chatPersist) string {
	var b strings.Builder
	if info, ok := providers.AsStreamClientInfo(client); ok {
		b.WriteString(info.ProviderKind())
		b.WriteString(" · ")
		b.WriteString(info.Model())
	} else {
		b.WriteString("provider unknown")
	}
	if persist != nil {
		b.WriteString(" · session ")
		b.WriteString(persist.store.ID)
	} else {
		b.WriteString(" · no disk session")
	}
	return b.String()
}

func tuiToolPreviewMax() int {
	const defaultMax = 4000
	s := strings.TrimSpace(os.Getenv("OPENCLAUDE_TUI_TOOL_PREVIEW"))
	if s == "" {
		return defaultMax
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 400
	}
	if n == 0 {
		return defaultMax
	}
	const capMax = 1 << 20
	if n > capMax {
		return capMax
	}
	return n
}

func tuiMarkdownEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("OPENCLAUDE_TUI_MARKDOWN")))
	return v != "0" && v != "false" && v != "no"
}

func mcpSummaryLine(mcp *mcpclient.Manager) string {
	if mcp == nil || len(mcp.Servers) == 0 {
		return ""
	}
	toolsN := 0
	for _, s := range mcp.Servers {
		toolsN += len(s.OpenAINames)
	}
	return fmt.Sprintf("MCP: %d tool(s) from %d server(s) — /mcp list", toolsN, len(mcp.Servers))
}

func printProviderInfo(c core.StreamClient) {
	printProviderInfoTo(c, os.Stdout)
}

func printProviderInfoTo(c core.StreamClient, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	info, ok := providers.AsStreamClientInfo(c)
	if !ok {
		_, _ = fmt.Fprintln(w, "(provider details unavailable)")
		return
	}
	base := info.BaseURL()
	if base == "" {
		base = "(default OpenAI API URL)"
	}
	_, _ = fmt.Fprintf(w, "Kind:    %s\nModel:   %s\nBase:    %s\nAPI key: %s\n",
		info.ProviderKind(), info.Model(), base, info.RedactedAPIKeySummary())
}

func printChatHelp() {
	printChatHelpTo(os.Stdout)
}

func printChatHelpTo(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	const help = `Commands:
  /onboard, /setup Quick env hints (same themes as doctor)
  /doctor       Same diagnostics as: openclaude doctor
  /config       Effective config summary (precedence, paths, provider/model/session/MCP names; no secrets)
  /permissions  Dangerous-tool auto-approve + MCP approval from config; see docs/SECURITY.md
  /version      Same as: openclaude version
  /init         Print starter openclaude.yaml snippet (see openclaude.example.yaml, docs/CONFIG.md)
  /export       Transcript export: /export [json|md] [path] or /export <path> (JSON to file)
  /context, /tokens  Rough token estimate + message count + compact settings
  /model [<id>] Show or set model for the active provider (updates session; TUI: not while busy)
  /provider     Show active provider, model, base URL, credential hint
  /provider wizard  Plain REPL: stdin wizard. TUI: opens $EDITOR on config file + YAML/env guide
  /provider <openai|ollama|gemini|github>  Switch provider (in-memory viper + new client)
  /provider show|status|help
  /mcp list    MCP tools connected in this process
  /mcp config  MCP servers from config file only (no subprocess)
  /mcp doctor  Same as list + tip: openclaude mcp doctor
  /mcp add     Print shell hint for openclaude mcp add ...
  /mcp help    Subcommands summary
  /session     Show active session path (when sessions enabled)
  /session list|load|new|save|running|ps  (see /session help via trying /session)
  /resume [<id>]  List saved sessions or load one (same as /session load)
  /skills list     List loaded skills
  /skills read <n> Print one skill body
  /<skill>     If name matches a loaded skill (case-insensitive), same as /skills read
  /btw <text>  Side question: one-shot answer, not added to main transcript
  /cost, /usage Transcript stats; billing not tracked in v4
  /copy        Copy last assistant message to clipboard (macOS/Linux when pbcopy/xclip/wl-copy exist)
  /theme light|dark|auto   TUI only: palette + markdown style
  /vim         TUI: reports vim-style input is not implemented
  /compact     Drop older messages (keeps system + tail; count from config)
  /clear       Clear conversation history for this session
  /help        Show this help
  /exit, /quit Exit

Tools: FileRead, FileWrite, FileEdit, Bash, Grep, Glob, WebSearch, WebFetch, GoOutline (Go AST outline), SkillsList, SkillsRead, SpiderScrape (only if spider CLI on PATH; no Firecrawl), Task (sub-agent), plus MCP tools (mcp_<server>__<tool>).
Vision: --image-url and --image-file attach to the first user message (REPL/TUI) or to -p one-shot; needs a vision-capable model.
Workspace is the current working directory.

Providers: openai (OPENAI_API_KEY), ollama (local), gemini (GEMINI_API_KEY or GOOGLE_API_KEY).
v3 users: .openclaude-profile.json in cwd or $HOME is merged automatically (under openclaude.yaml).
See docs/CONFIG.md and openclaude doctor.

Dangerous tools (including MCP tools with approval: ask) prompt unless OPENCLAUDE_AUTO_APPROVE_TOOLS=1

One-shot (non-interactive): openclaude -p "your question" prints only the final assistant reply on stdout
(streaming and tool traces go to stderr or are discarded). Use -p - to read the prompt from stdin.
`
	_, _ = fmt.Fprint(w, help)
}
