package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/session"
	sdk "github.com/sashabaranov/go-openai"
)

// errSlashExitChat signals the REPL should return (normal exit).
var errSlashExitChat = errors.New("slash exit chat")

const compactTailMessages = 24

type chatState struct {
	messages *[]sdk.ChatCompletionMessage
	mcpMgr   *mcpclient.Manager
	client   core.StreamClient
	persist  *chatPersist
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
		if st.persist != nil {
			_ = st.persist.Save()
		}
		_, _ = fmt.Fprintln(out, "(conversation cleared)")
	case "compact":
		cur := *st.messages
		next := compactTail(cur, compactTailMessages)
		if len(next) == len(cur) {
			_, _ = fmt.Fprintln(out, "(nothing to compact)")
			return nil
		}
		*st.messages = next
		if st.persist != nil {
			_ = st.persist.Save()
		}
		_, _ = fmt.Fprintf(out, "(compacted: kept system + last %d messages; older turns dropped)\n", compactTailMessages)
	case "session":
		return handleSessionSlash(args, st, out)
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

func handleSessionSlash(args []string, st chatState, out io.Writer) error {
	if st.persist == nil {
		return fmt.Errorf("sessions disabled — use --session <name>, --resume, or OPENCLAUDE_SESSION")
	}
	switch {
	case len(args) == 0 || args[0] == "show" || args[0] == "status":
		h := st.persist.handle
		n := 0
		if st.messages != nil {
			n = len(*st.messages)
		}
		_, _ = fmt.Fprintf(out, "Session %q → %s\n(%d messages in memory)\n", h.Name, h.Path(), n)
		return nil
	case args[0] == "list":
		list, err := session.List(st.persist.dir)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			_, _ = fmt.Fprintln(out, "(no saved sessions)")
			return nil
		}
		for _, e := range list {
			_, _ = fmt.Fprintf(out, "  %-20s  %d msgs  %s  cwd=%s\n", e.Name, e.NMsgs, e.Updated.Format(time.RFC3339), e.CWD)
		}
		return nil
	case args[0] == "save":
		if err := st.persist.Save(); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "(session saved)")
		return nil
	case args[0] == "load":
		if len(args) < 2 {
			return fmt.Errorf("usage: /session load <name>")
		}
		if err := st.persist.SwitchTo(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "(loaded session %q)\n", st.persist.handle.Name)
		return nil
	case args[0] == "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: /session new <name>")
		}
		if err := st.persist.BranchTo(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "(new session %q — transcript cleared)\n", st.persist.handle.Name)
		return nil
	default:
		return fmt.Errorf("unknown /session %q — try /session list, load, new, save, show", args[0])
	}
}

