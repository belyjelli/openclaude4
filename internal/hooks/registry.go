package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// SkillHookEvent names align with common v3 skill hook keys (subset).
type SkillHookEvent string

const (
	UserPromptSubmit SkillHookEvent = "UserPromptSubmit"
	PreToolUse       SkillHookEvent = "PreToolUse"
	PostToolUse      SkillHookEvent = "PostToolUse"
)

type commandHook struct {
	command string
	shell   string
	once    bool
}

type matcher struct {
	pattern string
	hooks   []commandHook
}

// Registry stores session-scoped hooks registered from SKILL.md frontmatter.
type Registry struct {
	mu       sync.Mutex
	sessions map[string]map[SkillHookEvent][]matcher // sessionID -> event -> matchers
	roots    map[string]string                       // sessionID -> last registered skill root (cwd for hooks)
}

var global = &Registry{
	sessions: make(map[string]map[SkillHookEvent][]matcher),
	roots:    make(map[string]string),
}

// Default returns the process-wide hook registry.
func Default() *Registry { return global }

// RegisterFromSkill parses v3-shaped hooks YAML and registers command hooks for the session.
func (r *Registry) RegisterFromSkill(sessionID, skillRoot string, raw map[string]any) error {
	if r == nil || sessionID == "" || len(raw) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions[sessionID] == nil {
		r.sessions[sessionID] = make(map[SkillHookEvent][]matcher)
	}
	if skillRoot != "" {
		r.roots[sessionID] = skillRoot
	}
	for evName, v := range raw {
		ev := SkillHookEvent(evName)
		if ev != UserPromptSubmit && ev != PreToolUse && ev != PostToolUse {
			continue
		}
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		for _, item := range arr {
			mobj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			pat, _ := mobj["matcher"].(string)
			hlist, _ := mobj["hooks"].([]any)
			var chs []commandHook
			for _, hi := range hlist {
				hm, ok := hi.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := hm["type"].(string); typ != "command" {
					continue
				}
				cmd, _ := hm["command"].(string)
				if strings.TrimSpace(cmd) == "" {
					continue
				}
				sh, _ := hm["shell"].(string)
				once, _ := hm["once"].(bool)
				chs = append(chs, commandHook{command: cmd, shell: strings.TrimSpace(strings.ToLower(sh)), once: once})
			}
			if len(chs) == 0 {
				continue
			}
			r.sessions[sessionID][ev] = append(r.sessions[sessionID][ev], matcher{pattern: pat, hooks: chs})
		}
	}
	return nil
}

// ClearSession removes all hooks for a session id.
func (r *Registry) ClearSession(sessionID string) {
	if r == nil || sessionID == "" {
		return
	}
	r.mu.Lock()
	delete(r.sessions, sessionID)
	delete(r.roots, sessionID)
	r.mu.Unlock()
}

// Dispatch runs all matching command hooks for an event. payload is JSON-encoded for the subprocess stdin when needed.
func (r *Registry) Dispatch(ctx context.Context, sessionID string, ev SkillHookEvent, skillRoot string, payload any) error {
	if r == nil || sessionID == "" {
		return nil
	}
	r.mu.Lock()
	matchers := append([]matcher(nil), r.sessions[sessionID][ev]...)
	root := r.roots[sessionID]
	r.mu.Unlock()
	if skillRoot == "" {
		skillRoot = root
	}
	if len(matchers) == 0 {
		return nil
	}
	data, _ := json.Marshal(payload)
	for _, m := range matchers {
		for _, h := range m.hooks {
			if err := runCommandHook(ctx, h, skillRoot, data); err != nil {
				return err
			}
		}
	}
	return nil
}

func runCommandHook(ctx context.Context, h commandHook, skillRoot string, stdinJSON []byte) error {
	cwd := skillRoot
	if cwd == "" {
		cwd = "."
	}
	cwd, _ = filepath.Abs(cwd)
	var cmd *exec.Cmd
	switch {
	case h.shell == "powershell" || h.shell == "pwsh":
		cmd = exec.CommandContext(ctx, "pwsh", "-NoProfile", "-Command", h.command)
	case runtime.GOOS == "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/C", h.command)
	default:
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", h.command)
	}
	cmd.Dir = cwd
	if skillRoot != "" {
		cmd.Env = append(os.Environ(), "CLAUDE_PLUGIN_ROOT="+skillRoot)
	}
	if len(stdinJSON) > 0 {
		cmd.Stdin = strings.NewReader(string(stdinJSON))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("skill hook: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
