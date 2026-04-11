package toolpolicy

import (
	"fmt"
	"strings"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/tools"
)

// Engine evaluates v3-style permission strings against a tool call.
type Engine struct {
	allow []string
	deny  []string
}

// NewEngine returns a policy engine with merged allow/deny lists (copied).
func NewEngine(allow, deny []string) *Engine {
	a := append([]string(nil), allow...)
	d := append([]string(nil), deny...)
	return &Engine{allow: a, deny: d}
}

// AppendAllow adds runtime allow rules (e.g. from UI); not persisted until [SessionStore.Append].
func (e *Engine) AppendAllow(rules []string) {
	for _, r := range rules {
		r = strings.TrimSpace(r)
		if r != "" {
			e.allow = append(e.allow, r)
		}
	}
}

// Eval returns a decided outcome, decided=true if policy resolves without user confirm, and a reason tag.
func (e *Engine) Eval(toolName string, args map[string]any) (out core.PermissionOutcome, decided bool, reason string) {
	if e == nil {
		return core.PermissionOutcome{}, false, ""
	}
	for _, rule := range e.deny {
		if matchRule(rule, toolName, args) {
			return core.DenyPermission(""), true, "policy_deny"
		}
	}
	for _, rule := range e.allow {
		if matchRule(rule, toolName, args) {
			return core.AllowPermission(), true, "policy_allow"
		}
	}
	return core.PermissionOutcome{}, false, ""
}

func matchRule(rule, toolName string, args map[string]any) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return false
	}
	if !strings.ContainsRune(rule, '(') {
		return strings.EqualFold(rule, toolName)
	}
	if strings.HasPrefix(strings.ToUpper(rule), "BASH(") && strings.HasSuffix(rule, ")") {
		if !strings.EqualFold(toolName, "Bash") {
			return false
		}
		inner := rule[5 : len(rule)-1]
		return matchBashInner(inner, args)
	}
	if strings.HasPrefix(rule, "FileWrite(") && strings.HasSuffix(rule, ")") {
		if !strings.EqualFold(toolName, "FileWrite") {
			return false
		}
		inner := strings.TrimSpace(rule[10 : len(rule)-1])
		return matchFilePath(inner, args)
	}
	if strings.HasPrefix(rule, "FileEdit(") && strings.HasSuffix(rule, ")") {
		if !strings.EqualFold(toolName, "FileEdit") {
			return false
		}
		inner := strings.TrimSpace(rule[9 : len(rule)-1])
		return matchFilePath(inner, args)
	}
	if strings.HasPrefix(toolName, "mcp_") {
		return strings.EqualFold(strings.TrimSpace(rule), toolName)
	}
	return strings.EqualFold(strings.TrimSpace(rule), toolName)
}

func matchFilePath(inner string, args map[string]any) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}
	p := argString(args, "file_path")
	if p == "" {
		return false
	}
	return matchPathPattern(inner, p)
}

func matchPathPattern(pattern, path string) bool {
	pattern = strings.TrimSpace(pattern)
	path = strings.TrimSpace(path)
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "*") {
		p := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, p)
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func matchBashInner(inner string, args map[string]any) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}
	cmd, _ := args["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	if strings.HasSuffix(inner, ":*") {
		prefix := strings.TrimSpace(strings.TrimSuffix(inner, ":*"))
		if prefix == "" {
			return false
		}
		fields := strings.Fields(cmd)
		if len(fields) == 0 {
			return false
		}
		if strings.EqualFold(fields[0], prefix) {
			return true
		}
		return strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(prefix+" "))
	}
	if strings.HasSuffix(inner, "*") {
		p := strings.TrimSuffix(inner, "*")
		return strings.HasPrefix(cmd, strings.TrimSpace(p))
	}
	return strings.EqualFold(cmd, inner)
}

func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

// SuggestedAllowRule returns a default allow string for UI prefill (best effort).
func SuggestedAllowRule(toolName string, args map[string]any) string {
	switch {
	case strings.EqualFold(toolName, "Bash"):
		cmd := argString(args, "command")
		fields := strings.Fields(cmd)
		if len(fields) > 0 {
			return "Bash(" + fields[0] + ":*)"
		}
		return "Bash"
	case strings.EqualFold(toolName, "FileWrite"), strings.EqualFold(toolName, "FileEdit"):
		p := argString(args, "file_path")
		if p == "" {
			return toolName
		}
		return toolName + "(" + p + "*)"
	default:
		return strings.TrimSpace(toolName)
	}
}

// BashDestructiveHint returns a one-line warning for obviously risky shell patterns, or "".
func BashDestructiveHint(command string) string {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return ""
	}
	patterns := []struct {
		substr, hint string
	}{
		{"rm -rf", "Warning: recursive delete (rm -rf)."},
		{"rm -r ", "Warning: recursive delete."},
		{"mkfs", "Warning: disk format (mkfs)."},
		{"dd if=", "Warning: block-level copy (dd) can destroy data."},
		{"shred ", "Warning: shredding overwrites files."},
		{"> /dev/", "Warning: writing to a device special file."},
		{"git push --force", "Warning: force push can overwrite remote history."},
		{"git push -f", "Warning: force push can overwrite remote history."},
		{"curl ", "Warning: curl may exfiltrate data or run untrusted payloads if piped to shell."},
		{"wget ", "Warning: wget may fetch and execute untrusted content if piped to shell."},
		{"chmod -R", "Warning: broad permission changes."},
		{"chown -R", "Warning: broad ownership changes."},
		{"sudo ", "Warning: elevated privileges."},
		{"curl|sh", "Warning: piping curl to shell is high risk."},
		{"curl | sh", "Warning: piping curl to shell is high risk."},
		{"wget|sh", "Warning: piping wget to shell is high risk."},
	}
	for _, p := range patterns {
		if strings.Contains(cmd, p.substr) {
			return p.hint
		}
	}
	// Reuse read-only classifier inverse signal: if not read-only allowlisted, nudge for shell metacharacters
	if strings.ContainsAny(cmd, "|&;`$()<>") && !tools.IsBashReadOnlyNoConfirm(command) {
		return "Warning: command uses shell metacharacters; review carefully."
	}
	return ""
}
