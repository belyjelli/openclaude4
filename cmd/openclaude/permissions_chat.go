package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/toolpolicy"
)

// replSessionAuto mirrors TUI session auto-approve for this stdin REPL process only.
var replSessionAuto atomic.Bool

// chatPermissionDeps builds the merged policy engine and optional session rule store.
func chatPermissionDeps(persist *chatPersist) (*toolpolicy.Engine, *toolpolicy.SessionStore) {
	allowG, denyG := config.PermissionsFromViper()
	merged := append([]string(nil), allowG...)
	var ps *toolpolicy.SessionStore
	if persist != nil && persist.store != nil && persist.store.Dir != "" && persist.store.ID != "" {
		p := toolpolicy.SessionPermissionsPath(persist.store.Dir, persist.store.ID)
		ps = toolpolicy.NewSessionStore(p)
		if extra, err := ps.Load(); err == nil {
			merged = append(merged, extra...)
		}
	}
	return toolpolicy.NewEngine(merged, denyG), ps
}

func replConfirm(r *bufio.Reader, toolName string, args map[string]any) core.PermissionOutcome {
	summary := core.FormatToolArgsForLog(args)
	if len(summary) > 400 {
		summary = summary[:397] + "..."
	}
	_, _ = fmt.Fprintf(os.Stderr, "\n--- permission required ---\nTool: %s\nArgs: %s\n", toolName, summary)
	_, _ = fmt.Fprintf(os.Stderr, "  [y] approve once   [a] session auto-approve on   [r] save allow rule + approve\n")
	_, _ = fmt.Fprintf(os.Stderr, "  [n] deny           [f] deny with note to model\n")
	_, _ = fmt.Fprintf(os.Stderr, "Choice: ")
	line, err := r.ReadString('\n')
	if err != nil {
		return core.DenyPermission("")
	}
	line = strings.TrimSpace(strings.ToLower(line))
	switch line {
	case "y", "yes":
		return core.AllowPermission()
	case "a":
		_, _ = fmt.Fprintln(os.Stderr, "(this process) auto-approve dangerous tools: on")
		return core.PermissionOutcome{Allow: true, EnableSessionAutoApprove: true}
	case "r":
		sug := toolpolicy.SuggestedAllowRule(toolName, args)
		_, _ = fmt.Fprintf(os.Stderr, "Allow rule [%s]: ", sug)
		ruleLine, err2 := r.ReadString('\n')
		if err2 != nil {
			return core.DenyPermission("")
		}
		rule := strings.TrimSpace(ruleLine)
		if rule == "" {
			rule = sug
		}
		return core.PermissionOutcome{Allow: true, AddAllowRules: []string{rule}}
	case "f":
		_, _ = fmt.Fprint(os.Stderr, "Note for model: ")
		note, err2 := r.ReadString('\n')
		if err2 != nil {
			return core.DenyPermission("")
		}
		return core.DenyPermission(strings.TrimSpace(note))
	case "n", "no":
		return core.DenyPermission("")
	default:
		return core.DenyPermission("")
	}
}
