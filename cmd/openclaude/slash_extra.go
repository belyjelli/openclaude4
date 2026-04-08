package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/skills"
	"github.com/gitlawb/openclaude4/internal/tui"
	sdk "github.com/sashabaranov/go-openai"
)

func printContextUsage(st chatState, out io.Writer) {
	var n, rough int
	if st.messages != nil {
		n = len(*st.messages)
		rough = session.RoughTokenEstimate(*st.messages)
	}
	th := config.SessionCompactTokenThreshold()
	keep := config.SessionCompactKeepMessages()
	_, _ = fmt.Fprintf(out, "Messages in memory: %d\nRough token estimate: %d (~4 chars per token)\n", n, rough)
	_, _ = fmt.Fprintf(out, "/compact keep last: %d messages (config session.compact_keep_messages)\n", keep)
	if th > 0 {
		_, _ = fmt.Fprintf(out, "Auto-compact token threshold: %d (OPENCLAUDE_SESSION_COMPACT_TOKEN_THRESHOLD)\n", th)
	} else {
		_, _ = fmt.Fprintln(out, "Auto-compact token threshold: off (set OPENCLAUDE_SESSION_COMPACT_TOKEN_THRESHOLD to enable)")
	}
}

func handleResumeSlash(args []string, st chatState, out io.Writer) error {
	if st.persist == nil {
		return fmt.Errorf("sessions disabled — omit --no-session to use /resume")
	}
	if len(args) == 0 {
		list, err := session.List(st.persist.dir)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			_, _ = fmt.Fprintln(out, "(no saved sessions)")
		} else {
			for _, e := range list {
				updated := ""
				if !e.Updated.IsZero() {
					updated = e.Updated.UTC().Format("2006-01-02T15:04:05Z")
				}
				_, _ = fmt.Fprintf(out, "  %-24s  %d msgs  %s  cwd=%s\n", e.Name, e.NMsgs, updated, e.CWD)
			}
		}
		_, _ = fmt.Fprintln(out, "Pick an id: /resume <id>  (same as /session load <id>)")
		return nil
	}
	id := strings.TrimSpace(args[0])
	if err := st.persist.SwitchTo(id); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "(loaded session %q)\n", st.persist.store.ID)
	return nil
}

func assistantContentString(m sdk.ChatCompletionMessage) string {
	if strings.TrimSpace(m.Content) != "" {
		return m.Content
	}
	var b strings.Builder
	for _, p := range m.MultiContent {
		if p.Type == sdk.ChatMessagePartTypeText {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func lastAssistantText(msgs []sdk.ChatCompletionMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != sdk.ChatMessageRoleAssistant {
			continue
		}
		return assistantContentString(msgs[i])
	}
	return ""
}

func slashCopyLastAssistant(st chatState, out io.Writer) error {
	if st.messages == nil {
		return fmt.Errorf("no transcript")
	}
	text := strings.TrimSpace(lastAssistantText(*st.messages))
	if text == "" {
		return fmt.Errorf("no assistant message to copy")
	}
	if copyToClipboard(text) {
		n := utf8.RuneCountInString(text)
		_, _ = fmt.Fprintf(out, "(copied last assistant reply, %d runes, to clipboard)\n", n)
		return nil
	}
	const maxShow = 800
	show := text
	if utf8.RuneCountInString(show) > maxShow {
		show = string([]rune(show)[:maxShow]) + "…"
	}
	_, _ = fmt.Fprintln(out, "Clipboard unavailable; paste from below:")
	_, _ = fmt.Fprintln(out, show)
	return nil
}

func slashCostOrUsage(st chatState, out io.Writer) {
	var n, rough int
	if st.messages != nil {
		n = len(*st.messages)
		rough = session.RoughTokenEstimate(*st.messages)
	}
	_, _ = fmt.Fprintf(out, "Session stats: %d messages, ~%d rough tokens in transcript.\n", n, rough)
	_, _ = fmt.Fprintln(out, "API dollar usage and billing are not tracked in OpenClaude v4.")
}

func slashBtw(st chatState, args []string, out io.Writer) error {
	if st.isBusy != nil && st.isBusy() {
		return fmt.Errorf("wait for the current model turn to finish before /btw")
	}
	q := strings.TrimSpace(strings.Join(args, " "))
	ctx := st.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ans, err := core.SideQuestion(ctx, effectiveClient(st), q)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "(btw — side question, not added to main transcript)")
	_, _ = fmt.Fprintln(out, ans)
	return nil
}

func slashTheme(st chatState, args []string, out io.Writer) error {
	if st.themeHolder == nil {
		_, _ = fmt.Fprintln(out, "/theme applies to TUI only (run with --tui).")
		return nil
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintf(out, "Current theme: %s (usage: /theme light|dark|auto)\n", st.themeHolder.Get())
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	switch mode {
	case "light", "dark", "auto":
		st.themeHolder.Set(mode)
		tui.ApplyTheme(mode)
		_, _ = fmt.Fprintf(out, "(theme set to %s)\n", mode)
		return nil
	default:
		return fmt.Errorf("unknown theme %q (use light, dark, auto)", args[0])
	}
}

func slashVim(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Vim-style input for the prompt is not implemented in v4 TUI (future work).")
}

func printSkillEntry(out io.Writer, e skills.Entry) {
	if e.Description != "" {
		_, _ = fmt.Fprintf(out, "# %s\n\n%s\n\n---\n\n", e.Name, e.Description)
	}
	_, _ = fmt.Fprint(out, e.Body)
	if !strings.HasSuffix(e.Body, "\n") {
		_, _ = fmt.Fprintln(out)
	}
}

func mcpAddShellHint() string {
	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		return "openclaude mcp add --name <id> --exec <argv>..."
	}
	return fmt.Sprintf("%s mcp add --name <id> --exec <argv>...  (repeat --exec per token)", exe)
}
