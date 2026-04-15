package bashv2

import "strings"

// CanonicalizeCommand strips full-line # comments (same behavior as legacy Bash tool).
// If every line would be removed, returns the original command unchanged.
func CanonicalizeCommand(command string) string {
	lines := strings.Split(command, "\n")
	var kept []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") {
			kept = append(kept, line)
		}
	}
	if len(kept) == 0 {
		return command
	}
	return strings.Join(kept, "\n")
}
