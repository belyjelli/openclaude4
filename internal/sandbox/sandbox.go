package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var blockedSubstrings = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs.",
	"dd if=",
	":(){",
	">& /dev/sd",
	"chmod -R 777 /",
}

// RunShell runs a shell one-liner with timeout and a basic safety filter.
func RunShell(ctx context.Context, command, cwd string) (string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("empty command")
	}
	lower := strings.ToLower(trimmed)
	for _, b := range blockedSubstrings {
		if strings.Contains(lower, strings.ToLower(b)) {
			return "", fmt.Errorf("command blocked by safety policy (matched %q)", b)
		}
	}

	var name string
	var args []string
	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/C", command}
	} else {
		name = "/bin/sh"
		args = []string{"-c", command}
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return strings.TrimSpace(out.String()), nil
}
