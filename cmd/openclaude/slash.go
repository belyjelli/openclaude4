package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/skills"
	sdk "github.com/sashabaranov/go-openai"
)

// errSlashExitChat signals the REPL should return (normal exit).
var errSlashExitChat = errors.New("slash exit chat")

type chatState struct {
	messages         *[]sdk.ChatCompletionMessage
	mcpMgr           *mcpclient.Manager
	client           core.StreamClient
	persist          *chatPersist
	providerWizardIn io.Reader // stdin in plain REPL; nil in TUI (wizard prints static guide only)
	skillCat         *skills.Catalog
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
	case "onboard", "setup":
		printOnboardHints(out)
	case "mcp":
		if len(args) > 0 && strings.EqualFold(args[0], "help") {
			printMCPHelp(out)
			return nil
		}
		if len(args) == 0 || args[0] == "list" {
			_, _ = fmt.Fprintln(out, st.mcpMgr.DescribeServers())
			return nil
		}
		if args[0] == "doctor" {
			_, _ = fmt.Fprintln(out, st.mcpMgr.DescribeServers())
			_, _ = fmt.Fprintln(out, "\nTip: for a fresh connect test from config (new processes), run: openclaude mcp doctor")
			return nil
		}
		return fmt.Errorf("unknown /mcp subcommand %q (try /mcp list, /mcp doctor, /mcp help)", args[0])
	case "clear":
		*st.messages = nil
		if st.persist != nil {
			_ = st.persist.Save()
		}
		_, _ = fmt.Fprintln(out, "(conversation cleared)")
	case "compact":
		keep := config.SessionCompactKeepMessages()
		cur := *st.messages
		next := session.CompactTail(cur, keep)
		if len(next) == len(cur) {
			_, _ = fmt.Fprintln(out, "(nothing to compact)")
			return nil
		}
		*st.messages = next
		if st.persist != nil {
			_ = st.persist.Save()
		}
		_, _ = fmt.Fprintf(out, "(compacted: kept system + last %d messages; older turns dropped)\n", keep)
	case "session":
		return handleSessionSlash(args, st, out)
	case "skills":
		return handleSkillsSlash(args, st, out)
	case "provider":
		if len(args) == 0 {
			printProviderInfoTo(st.client, out)
			return nil
		}
		sub := strings.ToLower(strings.TrimSpace(args[0]))
		switch sub {
		case "show", "status":
			printProviderInfoTo(st.client, out)
		case "wizard":
			return handleProviderWizard(st, out)
		case "help":
			_, _ = fmt.Fprint(out, `/provider              Show active provider, model, base URL, credential hint
/provider wizard      Step through setup (plain REPL only; restart openclaude to apply)
/provider show        Same as bare /provider
`)
		default:
			return fmt.Errorf("unknown /provider %q — try /provider wizard or /provider help", args[0])
		}
	default:
		return fmt.Errorf("unknown command %q - try /help", fields[0])
	}
	return nil
}

func handleSessionSlash(args []string, st chatState, out io.Writer) error {
	if len(args) > 0 {
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "running", "ps":
			return writeRunningList(out, config.EffectiveSessionDir())
		}
	}
	if st.persist == nil {
		return fmt.Errorf("sessions disabled — omit --no-session to enable on-disk sessions")
	}
	switch {
	case len(args) == 0 || args[0] == "show" || args[0] == "status":
		s := st.persist.store
		n := 0
		if st.messages != nil {
			n = len(*st.messages)
		}
		_, _ = fmt.Fprintf(out, "Session %q → %s\n(%d messages in memory)\n", s.ID, s.SessionPath(), n)
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
			updated := ""
			if !e.Updated.IsZero() {
				updated = e.Updated.UTC().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(out, "  %-24s  %d msgs  %s  cwd=%s\n", e.Name, e.NMsgs, updated, e.CWD)
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
		_, _ = fmt.Fprintf(out, "(loaded session %q)\n", st.persist.store.ID)
		return nil
	case args[0] == "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: /session new <name>")
		}
		if err := st.persist.BranchTo(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "(new session %q — transcript cleared)\n", st.persist.store.ID)
		return nil
	default:
		return fmt.Errorf("unknown /session %q — try /session list, running, load, new, save, show", args[0])
	}
}

func writeRunningList(w io.Writer, dir string) error {
	run, err := session.ListRunning(dir)
	if err != nil {
		return err
	}
	if len(run) == 0 {
		_, _ = fmt.Fprintf(w, "(no running-registry entries under %s/running)\n", dir)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Running registry (%s/running):\n", dir)
	for _, r := range run {
		st := "stale"
		if r.Alive {
			st = "alive"
		}
		mode := "repl"
		if r.Meta.TUI {
			mode = "tui"
		}
		pm := r.Meta.Provider
		if r.Meta.Model != "" {
			if pm != "" {
				pm = pm + "/" + r.Meta.Model
			} else {
				pm = r.Meta.Model
			}
		}
		_, _ = fmt.Fprintf(w, "  pid %-6d  %-5s  %-4s  session=%q  cwd=%s  started=%s\n",
			r.Meta.PID, st, mode, r.Meta.SessionID, r.Meta.CWD, r.Meta.Started)
		if pm != "" {
			_, _ = fmt.Fprintf(w, "            %s\n", pm)
		}
	}
	return nil
}

func handleSkillsSlash(args []string, st chatState, out io.Writer) error {
	cat := st.skillCat
	if cat == nil {
		var err error
		cat, err = skills.Load(config.SkillDirs())
		if err != nil {
			return fmt.Errorf("skills: %w", err)
		}
	}
	switch {
	case len(args) == 0 || args[0] == "list":
		if cat.Len() == 0 {
			_, _ = fmt.Fprintln(out, "(no skills loaded — add .openclaude/skills/<name>/SKILL.md or skills.dirs in config)")
			return nil
		}
		for _, e := range cat.List() {
			_, _ = fmt.Fprintf(out, "  %-24s  %s\n", e.Name, e.Description)
		}
		return nil
	case args[0] == "read":
		if len(args) < 2 {
			return fmt.Errorf("usage: /skills read <name>")
		}
		e, ok := cat.Get(args[1])
		if !ok {
			return fmt.Errorf("unknown skill %q", args[1])
		}
		if e.Description != "" {
			_, _ = fmt.Fprintf(out, "# %s\n\n%s\n\n---\n\n", e.Name, e.Description)
		}
		_, _ = fmt.Fprint(out, e.Body)
		if !strings.HasSuffix(e.Body, "\n") {
			_, _ = fmt.Fprintln(out)
		}
		return nil
	default:
		return fmt.Errorf("unknown /skills %q — try /skills list, /skills read <name>", args[0])
	}
}

func printMCPHelp(w io.Writer) {
	const text = `/mcp list    Tools from MCP servers connected in this process
/mcp doctor Same output plus a tip to run: openclaude mcp doctor
/mcp help   This text

Shell: openclaude mcp list | doctor | add
`
	_, _ = fmt.Fprint(w, text)
}

func printOnboardHints(w io.Writer) {
	const text = `Onboarding (see docs/CONFIG.md):
  openai   OPENAI_API_KEY  optional OPENAI_BASE_URL / OPENAI_MODEL
  ollama   OPENCLAUDE_PROVIDER=ollama  optional OLLAMA_HOST / OLLAMA_MODEL
  gemini   OPENCLAUDE_PROVIDER=gemini  GEMINI_API_KEY or GOOGLE_API_KEY

Verify: openclaude doctor
Saved + running processes: openclaude sessions
Full-screen UI: openclaude --tui
`
	_, _ = fmt.Fprint(w, text)
}
