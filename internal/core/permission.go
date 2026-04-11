package core

import "strings"

// PermissionOutcome is returned by [ConfirmTool] after policy and/or user confirmation.
type PermissionOutcome struct {
	Allow bool
	// DeclineUserNote is optional user text included in the tool result when Allow is false.
	DeclineUserNote string
	// EnableSessionAutoApprove requests the transport enable session-wide dangerous-tool auto-approve.
	EnableSessionAutoApprove bool
	// AddAllowRules are v3-style allow rule strings (e.g. "Bash(git:*)") for the transport to persist.
	AddAllowRules []string
}

// AllowPermission returns an allow outcome with no side effects.
func AllowPermission() PermissionOutcome {
	return PermissionOutcome{Allow: true}
}

// DenyPermission returns a deny outcome; note is optional feedback for the model.
func DenyPermission(note string) PermissionOutcome {
	return PermissionOutcome{
		Allow:           false,
		DeclineUserNote: strings.TrimSpace(note),
	}
}

// DeclineToolMessage builds the tool-result string sent to the model on deny.
func DeclineToolMessage(note string) string {
	n := strings.TrimSpace(note)
	if n == "" {
		return "User declined this tool execution."
	}
	return "User declined this tool execution.\n\n" + n
}

// PermissionPolicy, if set on [Agent], is run before prompting for dangerous tools.
// If decided is true, outcome is used and [Confirm] is not called. Reason tags: policy_allow, policy_deny.
type PermissionPolicy func(toolName string, args map[string]any) (outcome PermissionOutcome, decided bool, reason string)
