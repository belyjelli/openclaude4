package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List saved sessions and running local chat processes (registry)",
	Long: strings.TrimSpace(`
Shows on-disk session JSON files (same as --list-sessions) plus rows from
<session-dir>/running/<pid>.json written when a chat or TUI starts.
Stale rows remain until removed; "alive" uses a POSIX PID check where supported.`),
	RunE: runSessions,
}

func runSessions(*cobra.Command, []string) error {
	dir := config.EffectiveSessionDir()
	_, _ = fmt.Fprintf(os.Stdout, "Session directory: %s\n\n", dir)

	entries, err := session.List(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "(no saved session files)")
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Saved sessions:")
		for _, e := range entries {
			ts := "(unknown)"
			if !e.Updated.IsZero() {
				ts = e.Updated.UTC().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(os.Stdout, "  %-24s  %4d msgs  %s  cwd=%s\n", e.Name, e.NMsgs, ts, e.CWD)
		}
	}

	_, _ = fmt.Fprintln(os.Stdout)
	run, err := session.ListRunning(dir)
	if err != nil {
		return err
	}
	if len(run) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "(no running-registry entries — start chat or TUI to create one)")
		return nil
	}
	_, _ = fmt.Fprintln(os.Stdout, "Running registry (pid · alive · session · cwd · mode · provider/model):")
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
		if pm == "" {
			pm = "?"
		}
		if r.Meta.Model != "" {
			pm = pm + "/" + r.Meta.Model
		}
		_, _ = fmt.Fprintf(os.Stdout, "  pid %-6d  %-5s  session=%-20q  cwd=%s\n           mode=%s  %s  started=%s\n",
			r.Meta.PID, st, r.Meta.SessionID, r.Meta.CWD, mode, pm, r.Meta.Started)
	}
	return nil
}
