package tools

import (
	"strings"
)

// connectGlobalReadOnlyFlags: connect-cli globals (-c/--config, -v/--verbose) safe for read-only subcommands.
var connectGlobalReadOnlyFlags = map[string]ghFlagKind{
	"-v":        ghFlagNone,
	"--verbose": ghFlagNone,
	"-c":        ghFlagString,
	"--config":  ghFlagString,
}

// connectListReadOnlyFlags: connect list / ls plus globals (no connect / import / scan / cert / etc.).
var connectListReadOnlyFlags = joinGhFlags(
	connectGlobalReadOnlyFlags,
	map[string]ghFlagKind{
		"-d":         ghFlagNone,
		"--detailed": ghFlagNone,
	},
)

// IsConnectSafeReadOnlyCommand reports whether cmd is a read-only connect-cli invocation
// (list, show-config, or version only). When true, Bash may run without dangerous-tool confirmation.
func IsConnectSafeReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || bashCommandLooksCompound(cmd) {
		return false
	}
	words, err := shellSplit(cmd)
	if err != nil || len(words) < 2 {
		return false
	}
	if !isConnectExecutable(words[0]) {
		return false
	}
	if !connectArgsNoShellExpansion(words) {
		return false
	}
	i := 1
scanLeadingGlobals:
	for i < len(words) {
		w := words[i]
		switch {
		case w == "-v" || w == "--verbose":
			i++
		case w == "-c" || w == "--config":
			if i+1 >= len(words) {
				return false
			}
			i += 2
		case strings.HasPrefix(w, "--config="):
			i++
		default:
			break scanLeadingGlobals
		}
	}
	if i >= len(words) {
		return false
	}
	sub := strings.ToLower(words[i])
	rest := words[i+1:]
	switch sub {
	case "version":
		return validateConnectSubcommandFlags(connectGlobalReadOnlyFlags, rest)
	case "show-config", "showconfig":
		return validateConnectSubcommandFlags(connectGlobalReadOnlyFlags, rest)
	case "list", "ls":
		return validateConnectSubcommandFlags(connectListReadOnlyFlags, rest)
	default:
		return false
	}
}

func isConnectExecutable(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	base := tok
	if j := strings.LastIndexAny(tok, `/\`); j >= 0 {
		base = tok[j+1:]
	}
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".exe"), ".bat")
	return base == "connect"
}

func connectArgsNoShellExpansion(words []string) bool {
	for _, w := range words {
		if strings.Contains(w, "$") {
			return false
		}
	}
	return true
}

// validateConnectSubcommandFlags is like validateGhReadOnlyFlags but rejects stray positional tokens
// (connect list must not accept extra arguments).
func validateConnectSubcommandFlags(rule map[string]ghFlagKind, rest []string) bool {
	j := 0
	for j < len(rest) {
		t := rest[j]
		if !strings.HasPrefix(t, "-") {
			return false
		}
		name := t
		val := ""
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name, val = name[:eq], name[eq+1:]
		}
		kind, ok := rule[name]
		if !ok {
			return false
		}
		switch kind {
		case ghFlagNone:
			if val != "" {
				return false
			}
			j++
		case ghFlagString:
			if val == "" {
				if j+1 >= len(rest) {
					return false
				}
				j += 2
			} else {
				j++
			}
		case ghFlagNumber:
			if val == "" {
				if j+1 >= len(rest) {
					return false
				}
				val = rest[j+1]
				j += 2
			} else {
				j++
			}
			if !isGhNumericFlagArg(val) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
