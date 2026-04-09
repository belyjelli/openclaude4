package core

import (
	"os"
	"strings"
)

// EffectiveSystemPrompt returns [DefaultSystemPrompt] plus optional commit/PR attribution lines
// from OPENCLAUDE_COMMIT_ATTRIBUTION and OPENCLAUDE_PR_ATTRIBUTION (OpenClaude v3–style policy footers).
func EffectiveSystemPrompt() string {
	var b strings.Builder
	b.WriteString(DefaultSystemPrompt)
	if s := strings.TrimSpace(os.Getenv("OPENCLAUDE_COMMIT_ATTRIBUTION")); s != "" {
		b.WriteString("\n\n# Commit messages\nWhen creating git commits, append or include the following (user/org policy):\n")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if s := strings.TrimSpace(os.Getenv("OPENCLAUDE_PR_ATTRIBUTION")); s != "" {
		b.WriteString("\n\n# Pull request bodies\nWhen creating GitHub pull requests, include the following in the PR description (user/org policy):\n")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return b.String()
}
