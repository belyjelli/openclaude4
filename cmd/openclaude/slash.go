package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/chatlive"
	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/mcpclient"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/skills"
	"github.com/gitlawb/openclaude4/internal/tui"
	sdk "github.com/sashabaranov/go-openai"
)

// errSlashExitChat signals the REPL should return (normal exit).
var errSlashExitChat = errors.New("slash exit chat")

type chatState struct {
	messages                *[]sdk.ChatCompletionMessage
	mcpMgr                  *mcpclient.Manager
	client                  core.StreamClient
	live                    *chatlive.LiveChat
	persist                 *chatPersist
	providerWizardIn        io.Reader
	allowConfigEditorWizard bool
	skillCat                *skills.Catalog
	ctx                     context.Context
	isBusy                  func() bool
	themeHolder             *tui.ThemeHolder
	vimKeys                 *tui.VimKeysHolder
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
	case "doctor":
		PrintDoctorReport(out, version, commit)
	case "config":
		config.DescribeEffectiveConfig(out)
	case "permissions":
		printPermissionsSummary(out)
	case "version":
		printVersionSlash(out, version, commit)
	case "init":
		printInitSnippet(out)
	case "export":
		return slashExport(st, args, out)
	case "context", "tokens":
		printContextUsage(st, out)
	case "btw":
		return slashBtw(st, args, out)
	case "resume":
		return handleResumeSlash(args, st, out)
	case "model":
		return slashSetModel(st, strings.Join(args, " "), out)
	case "copy":
		return slashCopyLastAssistant(st, out)
	case "cost", "usage":
		slashCostOrUsage(st, out)
	case "theme":
		return slashTheme(st, args, out)
	case "vim":
		slashVim(st, out)
	case "mcp":
		if len(args) > 0 && strings.EqualFold(args[0], "help") {
			printMCPHelp(out)
			return nil
		}
		if len(args) > 0 && strings.EqualFold(args[0], "config") {
			PrintMCPConfigList(out)
			return nil
		}
		if len(args) > 0 && strings.EqualFold(args[0], "add") {
			_, _ = fmt.Fprintf(out, "Add servers from a shell (argv parsing):\n  %s\n", mcpAddShellHint())
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
		return fmt.Errorf("unknown /mcp subcommand %q (try /mcp list, config, doctor, add, help)", args[0])
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
			printProviderInfoTo(effectiveClient(st), out)
			return nil
		}
		sub := strings.ToLower(strings.TrimSpace(args[0]))
		switch sub {
		case "show", "status":
			printProviderInfoTo(effectiveClient(st), out)
		case "wizard":
			return handleProviderWizard(st, out)
		case "help":
			_, _ = fmt.Fprint(out, `/provider              Show active provider, model, base URL, credential hint
/provider wizard      Setup hints (stdin REPL) or open $EDITOR on config (TUI)
/provider show        Same as bare /provider
/provider <name>     Switch provider: openai | ollama | gemini | github | openrouter (in-memory; also sets provider.name)
`)
		case "openai", "ollama", "gemini", "github", "openrouter":
			return slashSetProvider(st, sub, out)
		default:
			return fmt.Errorf("unknown /provider %q — try /provider wizard, /provider <openai|ollama|gemini|github|openrouter>, or /provider help", args[0])
		}
	default:
		if st.skillCat != nil {
			if e, ok := st.skillCat.GetFold(cmd); ok {
				printSkillEntry(out, e)
				return nil
			}
		}
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
		printSkillEntry(out, e)
		return nil
	default:
		return fmt.Errorf("unknown /skills %q — try /skills list, /skills read <name>", args[0])
	}
}

func printMCPHelp(w io.Writer) {
	const text = `/mcp list    Tools from MCP servers connected in this process
/mcp config MCP servers as defined in config file (no subprocess)
/mcp doctor Same as list + tip to run: openclaude mcp doctor
/mcp add     Print shell hint to run: openclaude mcp add ...
/mcp help    This text

Shell: openclaude mcp list | doctor | add
`
	_, _ = fmt.Fprint(w, text)
}

func printOnboardHints(w io.Writer) {
	const text = `Onboarding (see docs/CONFIG.md):
  openai   OPENAI_API_KEY  optional OPENAI_BASE_URL / OPENAI_MODEL (OPENROUTER_KEY if base is OpenRouter)
  ollama   OPENCLAUDE_PROVIDER=ollama  optional OLLAMA_HOST / OLLAMA_MODEL
  gemini   OPENCLAUDE_PROVIDER=gemini  GEMINI_API_KEY or GOOGLE_API_KEY
  github   OPENCLAUDE_PROVIDER=github  GITHUB_TOKEN  optional GITHUB_BASE_URL / GITHUB_MODEL
  openrouter OPENCLAUDE_PROVIDER=openrouter  OPENROUTER_KEY  optional OPENROUTER_MODEL / OPENAI_BASE_URL

Verify: openclaude doctor
Saved + running processes: openclaude sessions
Full-screen UI: openclaude --tui
`
	_, _ = fmt.Fprint(w, text)
}
