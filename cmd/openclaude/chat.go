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
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
	"github.com/gitlawb/openclaude4/internal/session"
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
		case errors.Is(err, providers.ErrCodexNotImplemented):
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	wd, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	ctx = tools.WithWorkDir(ctx, wd)

	reg := tools.NewDefaultRegistry()
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
		return session.ApplyTokenThreshold(ctx, client, &messages,
			config.SessionCompactTokenThreshold(),
			config.SessionSummarizeOverThreshold(),
			config.SessionCompactKeepMessages(),
			core.DefaultSystemPrompt,
		)
	}

	autoApprove := strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "1") ||
		strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "true")

	useTUI := useTUIEarly || envTruthy("OPENCLAUDE_TUI")
	if printMode {
		if err := runPrintTurn(ctx, cmd, client, reg, &messages, beforeUserTurn, autoApprove, &agent); err != nil {
			return err
		}
		return nil
	}

	if useTUI {
		var banner strings.Builder
		writeChatBanner(&banner, client, mcpMgr)
		bannerStr := strings.TrimSpace(banner.String()) + "\nTUI: Ctrl+C to quit · same /commands as plain REPL."
		return tui.Run(tui.Config{
			Ctx:            ctx,
			Client:         client,
			Registry:       reg,
			Messages:       &messages,
			AutoApprove:    autoApprove,
			Banner:         bannerStr,
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
					messages:         &messages,
					mcpMgr:           mcpMgr,
					client:           client,
					persist:          persist,
					providerWizardIn: nil, // TUI: /provider wizard prints static guide only
				}, &out)
				if errors.Is(err, errSlashExitChat) {
					return out.String(), true, nil
				}
				return out.String(), false, err
			},
		})
	}

	printChatBanner(client, mcpMgr)

	reader := bufio.NewReader(os.Stdin)

	agent = &core.Agent{
		Client:   client,
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
				messages:         &messages,
				mcpMgr:           mcpMgr,
				client:           client,
				persist:          persist,
				providerWizardIn: os.Stdin,
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
		if err := agent.RunUserTurn(ctx, &messages, line); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
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
	if err := (*agent).RunUserTurn(ctx, messages, prompt); err != nil {
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

func printChatBanner(c core.StreamClient, mcp *mcpclient.Manager) {
	writeChatBanner(os.Stderr, c, mcp)
}

func writeChatBanner(w io.Writer, c core.StreamClient, mcp *mcpclient.Manager) {
	if info, ok := providers.AsStreamClientInfo(c); ok {
		_, _ = fmt.Fprintf(w, "OpenClaude v4 (phase 3). Provider: %s. Model: %s. Type /help. Ctrl+D to exit.\n",
			info.ProviderKind(), info.Model())
	} else {
		_, _ = fmt.Fprintln(w, "OpenClaude v4 (phase 3). Type /help. Ctrl+D to exit.")
	}
	if mcp != nil && len(mcp.Servers) > 0 {
		toolsN := 0
		for _, s := range mcp.Servers {
			toolsN += len(s.OpenAINames)
		}
		_, _ = fmt.Fprintf(w, "MCP: %d tool(s) from %d server(s) — /mcp list\n", toolsN, len(mcp.Servers))
	}
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
  /provider         Show active provider, model, base URL, credential hint
  /provider wizard  Interactive setup (YAML/env hints; plain REPL only — use /provider help)
  /mcp list    List connected MCP servers and tool names (see openclaude.yaml mcp.servers)
  /mcp doctor  Show same as list + tip to run openclaude mcp doctor for a fresh check
  /session     Show active session file path (when sessions enabled)
  /session list    List saved session files on disk
  /session load <id>   Switch to another session (saves current first)
  /session new <id>    New empty session under id (saves current first)
  /session save    Force write current transcript to disk
  /compact     Drop older messages (keeps system + last 24); lossy — use before long sessions
  /clear       Clear conversation history for this session
  /help        Show this help
  /exit        Exit (same as /quit)
  /quit        Exit

Tools: FileRead, FileWrite, FileEdit, Bash, Grep, Glob, WebSearch, WebFetch, SpiderScrape (only if spider CLI on PATH), Task (sub-agent), plus MCP tools (mcp_<server>__<tool>).
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
