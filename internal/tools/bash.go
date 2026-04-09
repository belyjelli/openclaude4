package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/gitlawb/openclaude4/internal/sandbox"
)

// Bash runs a shell command under the workspace with a timeout.
type Bash struct{}

func (Bash) Name() string      { return "Bash" }
func (Bash) IsDangerous() bool { return true }
func (Bash) Description() string {
	return "Run a shell command (sh -c on Unix, cmd /C on Windows). Optional cwd relative to workspace. Output is merged stdout+stderr. " +
		"For local repository work use git; for GitHub (issues, PRs, checks, releases, API-shaped data) use the gh CLI when available. " +
		"Read-only gh and docker commands that match the built-in allowlist (same idea as OpenClaude v3) can run without an extra dangerous-tool approval prompt."
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
				"description": "Timeout in seconds (default 120, max 600)",
			},
		},
		"required": []string{"command"},
	}
}

func (Bash) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmdStr, _ := args["command"].(string)
	if cmdStr == "" {
		return "", fmt.Errorf("command is required")
	}

	absRoot, err := resolveUnderWorkdir(ctx, ".")
	if err != nil {
		return "", err
	}
	cwd := absRoot
	if c, ok := args["cwd"].(string); ok && c != "" {
		cwd, err = resolveUnderWorkdir(ctx, c)
		if err != nil {
			return "", err
		}
	}

	sec := 120.0
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		sec = v
	}
	if sec > 600 {
		sec = 600
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(sec)*time.Second)
	defer cancel()

	return sandbox.RunShell(execCtx, cmdStr, cwd)
}
