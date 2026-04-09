package tools

import (
	"errors"
	"strings"
	"unicode"
)

// ssReadOnlyFlags mirrors OpenClaude v3 BashTool readOnlyValidation ss.safeFlags.
// Deliberately omits -K/--kill, -D/--diag, -F/--filter, -N/--net (see v3 comments).
var ssReadOnlyFlags = map[string]ghFlagKind{
	"-h":              ghFlagNone,
	"--help":          ghFlagNone,
	"-V":              ghFlagNone,
	"--version":       ghFlagNone,
	"-n":              ghFlagNone,
	"--numeric":       ghFlagNone,
	"-r":              ghFlagNone,
	"--resolve":       ghFlagNone,
	"-a":              ghFlagNone,
	"--all":           ghFlagNone,
	"-l":              ghFlagNone,
	"--listening":     ghFlagNone,
	"-o":              ghFlagNone,
	"--options":       ghFlagNone,
	"-e":              ghFlagNone,
	"--extended":      ghFlagNone,
	"-m":              ghFlagNone,
	"--memory":        ghFlagNone,
	"-p":              ghFlagNone,
	"--processes":     ghFlagNone,
	"-i":              ghFlagNone,
	"--info":          ghFlagNone,
	"-s":              ghFlagNone,
	"--summary":       ghFlagNone,
	"-4":              ghFlagNone,
	"--ipv4":          ghFlagNone,
	"-6":              ghFlagNone,
	"--ipv6":          ghFlagNone,
	"-0":              ghFlagNone,
	"--packet":        ghFlagNone,
	"-t":              ghFlagNone,
	"--tcp":           ghFlagNone,
	"-M":              ghFlagNone,
	"--mptcp":         ghFlagNone,
	"-S":              ghFlagNone,
	"--sctp":          ghFlagNone,
	"-u":              ghFlagNone,
	"--udp":           ghFlagNone,
	"-d":              ghFlagNone,
	"--dccp":          ghFlagNone,
	"-w":              ghFlagNone,
	"--raw":           ghFlagNone,
	"-x":              ghFlagNone,
	"--unix":          ghFlagNone,
	"--tipc":          ghFlagNone,
	"--vsock":         ghFlagNone,
	"-f":              ghFlagString,
	"--family":        ghFlagString,
	"-A":              ghFlagString,
	"--query":         ghFlagString,
	"--socket":        ghFlagString,
	"-Z":              ghFlagNone,
	"--context":       ghFlagNone,
	"-z":              ghFlagNone,
	"--contexts":      ghFlagNone,
	"-b":              ghFlagNone,
	"--bpf":           ghFlagNone,
	"-E":              ghFlagNone,
	"--events":        ghFlagNone,
	"-H":              ghFlagNone,
	"--no-header":     ghFlagNone,
	"-O":              ghFlagNone,
	"--oneline":       ghFlagNone,
	"--tipcinfo":      ghFlagNone,
	"--tos":           ghFlagNone,
	"--cgroup":        ghFlagNone,
	"--inet-sockopt":  ghFlagNone,
}

// IsSsSafeReadOnlyCommand reports whether cmd is a read-only ss (iproute2) invocation matching
// OpenClaude v3's ss allowlist. When true, Bash may run without dangerous-tool confirmation.
func IsSsSafeReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || bashCommandLooksCompound(cmd) {
		return false
	}
	words, err := shellSplit(cmd)
	if err != nil || len(words) < 1 {
		return false
	}
	if !isSsExecutable(words[0]) {
		return false
	}
	rest, err := expandSsShortFlagBundles(words[1:])
	if err != nil {
		return false
	}
	return validateGhReadOnlyFlags(ssReadOnlyFlags, rest)
}

func isSsExecutable(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	base := tok
	if i := strings.LastIndexAny(tok, `/\`); i >= 0 {
		base = tok[i+1:]
	}
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".exe"), ".bat")
	return base == "ss"
}

func expandSsShortFlagBundles(args []string) ([]string, error) {
	var out []string
	for _, t := range args {
		if t == "--" {
			return nil, errors.New("ss: -- not allowed in read-only validator")
		}
		if isSsBundlableShortToken(t) {
			expanded := expandOneSsBundle(t)
			if expanded != nil {
				out = append(out, expanded...)
				continue
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func isSsBundlableShortToken(t string) bool {
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

func expandOneSsBundle(t string) []string {
	for _, r := range t[1:] {
		sf := "-" + string(r)
		kind, ok := ssReadOnlyFlags[sf]
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
