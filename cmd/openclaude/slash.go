package main

import (
	"errors"
	"fmt"
	"io"
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

func handleSlashLine(line string, st chatState, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
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
		printChatHelpTo(out)
	case "mcp":
		if len(args) == 0 || args[0] == "list" {
			_, _ = fmt.Fprintln(out, st.mcpMgr.DescribeServers())
			return nil
		}
		return fmt.Errorf("unknown /mcp subcommand %q (try /mcp list)", args[0])
	case "clear":
		*st.messages = nil
		_, _ = fmt.Fprintln(out, "(conversation cleared)")
	case "compact":
		cur := *st.messages
		next := compactTail(cur, compactTailMessages)
		if len(next) == len(cur) {
			_, _ = fmt.Fprintln(out, "(nothing to compact)")
			return nil
		}
		*st.messages = next
		_, _ = fmt.Fprintf(out, "(compacted: kept system + last %d messages; older turns dropped)\n", compactTailMessages)
	case "provider":
		printProviderInfoTo(st.client, out)
	default:
		return fmt.Errorf("unknown command %q - try /help", fields[0])
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

