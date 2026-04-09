package tools

import (
	"strconv"
	"strings"
)

type ghFlagKind byte

const (
	ghFlagNone ghFlagKind = iota
	ghFlagString
	ghFlagNumber
)

// IsGHSafeReadOnlyCommand reports whether cmd is a read-only GitHub CLI invocation that
// matches OpenClaude v3's GH_READ_ONLY_COMMANDS allowlist (same repo/URL exfil guards).
// When true, the agent may run Bash without a dangerous-tool confirmation (subject to policy).
func IsGHSafeReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || bashCommandLooksCompound(cmd) {
		return false
	}
	words, err := shellSplit(cmd)
	if err != nil || len(words) < 3 {
		return false
	}
	if !isGhExecutable(words[0]) {
		return false
	}
	if ghArgsHaveSuspiciousRepoOrURL(words[1:]) {
		return false
	}

	i := 1
	for i < len(words) {
		w := words[i]
		if w == "-R" || w == "--repo" {
			if i+1 >= len(words) {
				return false
			}
			i += 2
			continue
		}
		if strings.HasPrefix(w, "--repo=") {
			i++
			continue
		}
		break
	}
	if i+2 > len(words) {
		return false
	}
	key := words[i] + " " + words[i+1]
	rule, ok := ghReadOnlySubcommands[key]
	if !ok {
		return false
	}
	rest := words[i+2:]
	return validateGhReadOnlyFlags(rule, rest)
}

func isGhNumericFlagArg(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func validateGhReadOnlyFlags(rule map[string]ghFlagKind, rest []string) bool {
	j := 0
	for j < len(rest) {
		t := rest[j]
		if strings.HasPrefix(t, "-") {
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
			}
			continue
		}
		j++
	}
	return true
}
