package tools

import (
	"errors"
	"strings"
	"unicode"
)

// netstatReadOnlyFlags mirrors OpenClaude v3 BashTool readOnlyValidation netstat.safeFlags.
var netstatReadOnlyFlags = map[string]ghFlagKind{
	"-a": ghFlagNone,
	"-L": ghFlagNone,
	"-l": ghFlagNone,
	"-n": ghFlagNone,
	"-f": ghFlagString,
	"-g": ghFlagNone,
	"-i": ghFlagNone,
	"-I": ghFlagString,
	"-s": ghFlagNone,
	"-r": ghFlagNone,
	"-m": ghFlagNone,
	"-v": ghFlagNone,
}

// IsNetstatSafeReadOnlyCommand reports whether cmd is a read-only netstat invocation matching
// OpenClaude v3's netstat allowlist. When true, Bash may run without dangerous-tool confirmation.
func IsNetstatSafeReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || bashCommandLooksCompound(cmd) {
		return false
	}
	words, err := shellSplit(cmd)
	if err != nil || len(words) < 1 {
		return false
	}
	if !isNetstatExecutable(words[0]) {
		return false
	}
	rest, err := expandNetstatShortFlagBundles(words[1:])
	if err != nil {
		return false
	}
	return validateGhReadOnlyFlags(netstatReadOnlyFlags, rest)
}

func isNetstatExecutable(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	base := tok
	if i := strings.LastIndexAny(tok, `/\`); i >= 0 {
		base = tok[i+1:]
	}
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".exe"), ".bat")
	return base == "netstat"
}

// expandNetstatShortFlagBundles splits BSD-style combined short flags (e.g. -an → -a, -n)
// when every letter is an allowlisted no-arg flag, matching v3 validateFlags bundling rules.
func expandNetstatShortFlagBundles(args []string) ([]string, error) {
	var out []string
	for _, t := range args {
		if t == "--" {
			return nil, errors.New("netstat: -- not allowed in read-only validator")
		}
		if isNetstatBundlableShortToken(t) {
			expanded := expandOneNetstatBundle(t)
			if expanded != nil {
				out = append(out, expanded...)
				continue
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func isNetstatBundlableShortToken(t string) bool {
	if !strings.HasPrefix(t, "-") || strings.HasPrefix(t, "--") || len(t) <= 2 {
		return false
	}
	for _, r := range t[1:] {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func expandOneNetstatBundle(t string) []string {
	for _, r := range t[1:] {
		sf := "-" + string(r)
		kind, ok := netstatReadOnlyFlags[sf]
		if !ok || kind != ghFlagNone {
			return nil
		}
	}
	out := make([]string, 0, len(t)-1)
	for _, r := range t[1:] {
		out = append(out, "-"+string(r))
	}
	return out
}

// IsBashReadOnlyNoConfirm is true when cmd matches a v3-style read-only Bash allowlist entry
// (currently gh and netstat) so the agent can skip the dangerous-tool confirmation prompt.
func IsBashReadOnlyNoConfirm(cmd string) bool {
	return IsGHSafeReadOnlyCommand(cmd) || IsNetstatSafeReadOnlyCommand(cmd)
}
