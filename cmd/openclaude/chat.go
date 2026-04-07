package main

import (
	"bufio"
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
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
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

	printChatBanner(client, mcpMgr)

	var messages []sdk.ChatCompletionMessage
	reader := bufio.NewReader(os.Stdin)

	autoApprove := strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "1") ||
		strings.EqualFold(os.Getenv("OPENCLAUDE_AUTO_APPROVE_TOOLS"), "true")

	agent := &core.Agent{
		Client:   client,
		Registry: reg,
		Out:      os.Stdout,
		Confirm: func(toolName string, args map[string]any) bool {
			if autoApprove {
				_, _ = fmt.Fprintf(os.Stderr, "[auto-approved] %s\n", toolName)
				return true
			}
			_, _ = fmt.Fprintf(os.Stderr, "Approve tool %q with args %v? [y/N]: ", toolName, args)
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

		switch {
		case line == "/exit" || line == "/quit":
			return nil
		case line == "/help":
			printChatHelp()
			continue
		case line == "/mcp" || line == "/mcp list":
			_, _ = fmt.Fprintln(os.Stdout, mcpMgr.DescribeServers())
			continue
		case line == "/clear":
			messages = nil
			_, _ = fmt.Fprintln(os.Stdout, "(conversation cleared)")
			continue
		case line == "/provider":
			printProviderInfo(client)
			continue
		case strings.HasPrefix(line, "/"):
			_, _ = fmt.Fprintf(os.Stderr, "Unknown command %q. Try /help.\n", strings.Fields(line)[0])
			continue
		}

		if err := agent.RunUserTurn(ctx, &messages, line); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
	}
}

func printChatBanner(c core.StreamClient, mcp *mcpclient.Manager) {
	if info, ok := providers.AsStreamClientInfo(c); ok {
		_, _ = fmt.Fprintf(os.Stderr, "OpenClaude v4 (phase 3). Provider: %s. Model: %s. Type /help. Ctrl+D to exit.\n",
			info.ProviderKind(), info.Model())
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "OpenClaude v4 (phase 3). Type /help. Ctrl+D to exit.")
	}
	if mcp != nil && len(mcp.Servers) > 0 {
		toolsN := 0
		for _, s := range mcp.Servers {
			toolsN += len(s.OpenAINames)
		}
		_, _ = fmt.Fprintf(os.Stderr, "MCP: %d tool(s) from %d server(s) — /mcp list\n", toolsN, len(mcp.Servers))
	}
}

func printProviderInfo(c core.StreamClient) {
	info, ok := providers.AsStreamClientInfo(c)
	if !ok {
		_, _ = fmt.Fprintln(os.Stdout, "(provider details unavailable)")
		return
	}
	base := info.BaseURL()
	if base == "" {
		base = "(default OpenAI API URL)"
	}
	_, _ = fmt.Fprintf(os.Stdout, "Kind:    %s\nModel:   %s\nBase:    %s\nAPI key: %s\n",
		info.ProviderKind(), info.Model(), base, info.RedactedAPIKeySummary())
}

func printChatHelp() {
	const help = `Commands:
  /provider    Show active provider, model, base URL, credential hint
  /mcp list    List connected MCP servers and tool names (see openclaude.yaml mcp.servers)
  /clear       Clear conversation history for this session
  /help        Show this help
  /exit        Exit (same as /quit)
  /quit        Exit

Tools: FileRead, FileWrite, FileEdit, Bash, Grep, Glob, WebSearch, plus MCP tools (mcp_<server>__<tool>).
Workspace is the current working directory.

Providers: openai (OPENAI_API_KEY), ollama (local), gemini (GEMINI_API_KEY or GOOGLE_API_KEY).
v3 users: .openclaude-profile.json in cwd or $HOME is merged automatically (under openclaude.yaml).
See docs/CONFIG.md and openclaude doctor.

Dangerous tools (including MCP tools with approval: ask) prompt unless OPENCLAUDE_AUTO_APPROVE_TOOLS=1
`
	_, _ = fmt.Fprint(os.Stdout, help)
}
