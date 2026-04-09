package ghstatus

import (
	"context"
	"os/exec"
	"time"
)

// GhAuthSummary reports whether gh is on PATH and whether `gh auth token` succeeds (local auth, no network).
func GhAuthSummary(ctx context.Context) (installed, authenticated bool) {
	if _, err := exec.LookPath("gh"); err != nil {
		return false, false
	}
	installed = true
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "gh", "auth", "token")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return installed, false
	}
	return installed, true
}
