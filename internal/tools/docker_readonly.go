package tools

import "strings"

// dockerLogsReadOnlyFlags mirrors OpenClaude v3 src/utils/shell/readOnlyCommandValidation.ts
// DOCKER_READ_ONLY_COMMANDS["docker logs"].safeFlags.
var dockerLogsReadOnlyFlags = map[string]ghFlagKind{
	"--follow":     ghFlagNone,
	"-f":           ghFlagNone,
	"--tail":       ghFlagString,
	"-n":           ghFlagString,
	"--timestamps": ghFlagNone,
	"-t":           ghFlagNone,
	"--since":      ghFlagString,
	"--until":      ghFlagString,
	"--details":    ghFlagNone,
}

// dockerInspectReadOnlyFlags mirrors v3 DOCKER_READ_ONLY_COMMANDS["docker inspect"].safeFlags.
var dockerInspectReadOnlyFlags = map[string]ghFlagKind{
	"--format": ghFlagString,
	"-f":       ghFlagString,
	"--type":   ghFlagString,
	"--size":   ghFlagNone,
	"-s":       ghFlagNone,
}

// IsDockerSafeReadOnlyCommand reports whether cmd is a read-only docker invocation matching
// OpenClaude v3's docker EXTERNAL_READONLY_COMMANDS + DOCKER_READ_ONLY_COMMANDS. When true,
// Bash may run without dangerous-tool confirmation.
func IsDockerSafeReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || bashCommandLooksCompound(cmd) {
		return false
	}
	words, err := shellSplit(cmd)
	if err != nil || len(words) < 2 {
		return false
	}
	if !isDockerExecutable(words[0]) {
		return false
	}
	for _, w := range words {
		if strings.Contains(w, "$") {
			return false
		}
	}
	sub := strings.ToLower(words[1])
	switch sub {
	case "ps", "images":
		return true
	case "logs":
		return validateGhReadOnlyFlags(dockerLogsReadOnlyFlags, words[2:])
	case "inspect":
		return validateGhReadOnlyFlags(dockerInspectReadOnlyFlags, words[2:])
	default:
		return false
	}
}

func isDockerExecutable(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	base := tok
	if i := strings.LastIndexAny(tok, `/\`); i >= 0 {
		base = tok[i+1:]
	}
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".exe"), ".bat")
	return base == "docker"
}
