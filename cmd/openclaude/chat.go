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
	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
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
	if persist != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Session %q — %d message(s) loaded · %s\n",
			persist.handle.Name, len(messages), persist.handle.Path())
	}

	autoApprove := strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "1") ||
		strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "true")

	useTUI, _ := cmd.Flags().GetBool("tui")
	if useTUI || envTruthy("OPENCLAUDE_TUI") {
		var banner strings.Builder
		writeChatBanner(&banner, client, mcpMgr)
		bannerStr := strings.TrimSpace(banner.String()) + "\nTUI: Ctrl+C to quit · same /commands as plain REPL."
		return tui.Run(tui.Config{
			Ctx:         ctx,
			Client:      client,
			Registry:    reg,
			Messages:    &messages,
			AutoApprove: autoApprove,
			Banner:      bannerStr,
			AfterTurn: func() error {
				if persist != nil {
					return persist.Save()
				}
				return nil
			},
			Slash: func(line string) (string, bool, error) {
				var out bytes.Buffer
				err := handleSlashLine(line, chatState{
					messages: &messages,
					mcpMgr:   mcpMgr,
					client:   client,
					persist:  persist,
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
				messages: &messages,
				mcpMgr:   mcpMgr,
				client:   client,
				persist:  persist,
			}, os.Stdout)
			if errors.Is(err, errSlashExitChat) {
				return nil
			}
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
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

// chatPersist binds a named session file to the in-memory transcript.
type chatPersist struct {
	dir      string
	wd       string
	handle   *session.Handle
	messages *[]sdk.ChatCompletionMessage
}

func (p *chatPersist) Save() error {
	if p == nil || p.handle == nil || p.messages == nil {
		return nil
	}
	return p.handle.SaveFrom(*p.messages, p.wd)
}

func (p *chatPersist) SwitchTo(name string) error {
	if p == nil {
		return fmt.Errorf("sessions not enabled")
	}
	h, err := session.NewHandle(p.dir, name)
	if err != nil {
		return err
	}
	if p.handle != nil {
		if err := p.Save(); err != nil {
			return fmt.Errorf("save before switch: %w", err)
		}
	}
	p.handle = h
	*p.messages = nil
	return h.LoadInto(p.messages)
}

func (p *chatPersist) BranchTo(name string) error {
	if p == nil {
		return fmt.Errorf("sessions not enabled")
	}
	h, err := session.NewHandle(p.dir, name)
	if err != nil {
		return err
	}
	if p.handle != nil {
		if err := p.Save(); err != nil {
			return fmt.Errorf("save before switch: %w", err)
		}
	}
	p.handle = h
	*p.messages = nil
	return p.Save()
}

func resolveChatPersistence(cmd *cobra.Command, wd string, messages *[]sdk.ChatCompletionMessage) (*chatPersist, error) {
	resume, _ := cmd.Flags().GetBool("resume")
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
	dir, err := session.DefaultDir()
	if err != nil {
		return nil, err
	}
	var sessName string
	if resume {
		n, err := session.LatestName(dir)
		if err != nil {
			return nil, err
		}
		sessName = n
	} else if name != "" {
		sessName = name
	} else {
		return nil, nil
	}
	h, err := session.NewHandle(dir, sessName)
	if err != nil {
		return nil, err
	}
	p := &chatPersist{dir: dir, wd: wd, handle: h, messages: messages}
	if err := h.LoadInto(messages); err != nil {
		return nil, err
	}
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
  /provider    Show active provider, model, base URL, credential hint
  /mcp list    List connected MCP servers and tool names (see openclaude.yaml mcp.servers)
  /compact     Drop older messages (keeps system + last 24); lossy — use before long sessions
  /clear       Clear conversation history for this session
  /help        Show this help
  /exit        Exit (same as /quit)
  /quit        Exit

Tools: FileRead, FileWrite, FileEdit, Bash, Grep, Glob, WebSearch, Task (sub-agent), plus MCP tools (mcp_<server>__<tool>).
Workspace is the current working directory.

Providers: openai (OPENAI_API_KEY), ollama (local), gemini (GEMINI_API_KEY or GOOGLE_API_KEY).
v3 users: .openclaude-profile.json in cwd or $HOME is merged automatically (under openclaude.yaml).
See docs/CONFIG.md and openclaude doctor.

Dangerous tools (including MCP tools with approval: ask) prompt unless OPENCLAUDE_AUTO_APPROVE_TOOLS=1
`
	_, _ = fmt.Fprint(w, help)
}
