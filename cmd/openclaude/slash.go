package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	sdk "github.com/sashabaranov/go-openai"
)

// errSlashExitChat signals the REPL should return (normal exit).
var errSlashExitChat = errors.New("slash exit chat")

const compactTailMessages = 24

type chatState struct {
	messages *[]sdk.ChatCompletionMessage
	mcpMgr   *mcpclient.Manager
	client   core.StreamClient
}

func handleSlashLine(line string, st chatState) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	cmd := strings.TrimPrefix(fields[0], "/")
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	args := fields[1:]

	switch cmd {
	case "exit", "quit":
		return errSlashExitChat
	case "help":
		printChatHelp()
	case "mcp":
		if len(args) == 0 || args[0] == "list" {
			_, _ = fmt.Fprintln(os.Stdout, st.mcpMgr.DescribeServers())
			return nil
		}
		return fmt.Errorf("unknown /mcp subcommand %q (try /mcp list)", args[0])
	case "clear":
		*st.messages = nil
		_, _ = fmt.Fprintln(os.Stdout, "(conversation cleared)")
	case "compact":
		n := compactTail(*st.messages, compactTailMessages)
		if n == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "(nothing to compact)")
			return nil
		}
		*st.messages = n
		_, _ = fmt.Fprintf(os.Stdout, "(compacted: kept system + last %d messages; older turns dropped)\n", compactTailMessages)
	case "provider":
		printProviderInfo(st.client)
	default:
		return fmt.Errorf("unknown command %q — try /help", fields[0])
	}
	return nil
}

// compactTail keeps the first system message (if present) and the last maxAfterSystem messages.
func compactTail(msgs []sdk.ChatCompletionMessage, maxAfterSystem int) []sdk.ChatCompletionMessage {
	if len(msgs) == 0 {
		return nil
	}
	if maxAfterSystem < 1 {
		maxAfterSystem = 1
	}
	if len(msgs) > 0 && msgs[0].Role == sdk.ChatMessageRoleSystem {
		sys := msgs[0]
		rest := msgs[1:]
		if len(rest) <= maxAfterSystem {
			return msgs
		}
		out := make([]sdk.ChatCompletionMessage, 0, 1+maxAfterSystem)
		out = append(out, sys)
		out = append(out, rest[len(rest)-maxAfterSystem:]...)
		return out
	}
	if len(msgs) <= maxAfterSystem {
		return msgs
	}
	return msgs[len(msgs)-maxAfterSystem:]
}

