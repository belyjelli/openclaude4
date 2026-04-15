package tools

import (
	"context"
	"fmt"

	"github.com/gitlawb/openclaude4/internal/bashv2"
)

// Bash runs a shell command under the workspace with the bash v2 pipeline (snapshot, sandbox, output shaping).
type Bash struct{}

func (Bash) Name() string      { return "Bash" }
func (Bash) IsDangerous() bool { return true }
func (Bash) Description() string {
	return "Run a shell command (bash v2: validated, sandboxed on Linux with bubblewrap when available, merged stdout+stderr). " +
		"Optional cwd relative to workspace. Read-only gh/docker/connect/ss/netstat patterns may auto-approve when policy allows. " +
		"Configure under `bashv2:` in openclaude.yaml."
}

func (Bash) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Full shell command string",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory relative to workspace (default: workspace root)",
			},
			"timeout_seconds": map[string]any{
				"type":        "number",
				"description": "Timeout in seconds (default 120, max from bashv2.maxTimeoutSeconds)",
			},
		},
		"required": []string{"command"},
	}
}

func (Bash) Execute(ctx context.Context, args map[string]any) (string, error) {
	s := bashv2.FromContext(ctx)
	if s == nil {
		return "", fmt.Errorf("Bash tool requires a bash v2 session on the context (internal wiring error)")
	}
	absRoot, err := resolveUnderWorkdir(ctx, ".")
	if err != nil {
		return "", err
	}
	return s.Execute(ctx, bashv2.ExecuteInput{
		ToolCallID: ToolCallID(ctx),
		Args:       args,
		Workspace:  absRoot,
	})
}
