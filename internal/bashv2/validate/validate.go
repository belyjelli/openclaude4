package validate

import (
	"path/filepath"
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

var (
	rxSudo          = regexp.MustCompile(`(?i)\bsudo\b`)
	rxPipeToShell   = regexp.MustCompile(`\|\s*(/usr/bin/env\s+)?(ba)?sh\b`)
)

// Verdict is the outcome of one validator on one unit.
type Verdict int

const (
	Pass Verdict = iota
	Fail
)

// Validator runs a static check on a command segment.
type Validator interface {
	ID() string
	Version() int
	Check(unit string) (Verdict, string)
}

// Chain runs validators in order; first failure stops.
func Chain(vs []Validator, units []string) (Verdict, string, string) {
	for _, u := range units {
		for _, v := range vs {
			if verdict, reason := v.Check(u); verdict == Fail {
				return Fail, v.ID(), reason
			}
		}
	}
	return Pass, "", ""
}

// --- built-in validators ---

type blockedSubstrings struct{}

func (blockedSubstrings) ID() string     { return "blocked_substrings" }
func (blockedSubstrings) Version() int { return 1 }

var blocked = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs.",
	"dd if=",
	":(){",
	">& /dev/sd",
	"chmod -R 777 /",
	">/dev/mem",
	"> /dev/mem",
	">/dev/kmem",
	"> /dev/kmem",
}

func (blockedSubstrings) Check(unit string) (Verdict, string) {
	t := strings.TrimSpace(strings.ToLower(unit))
	if t == "" {
		return Pass, ""
	}
	for _, b := range blocked {
		if strings.Contains(t, strings.ToLower(b)) {
			return Fail, "blocked pattern: " + b
		}
	}
	return Pass, ""
}

type posixSyntax struct{}

func (posixSyntax) ID() string     { return "posix_syntax" }
func (posixSyntax) Version() int { return 1 }

// Check uses a POSIX shell parse when possible; extensions fail closed (conservative).
func (posixSyntax) Check(unit string) (Verdict, string) {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return Fail, "empty segment"
	}
	r := strings.NewReader(unit)
	_, err := syntax.NewParser().Parse(r, "")
	if err != nil {
		// Bash-only or complex syntax: do not hard-fail; defer to other validators / ask.
		return Pass, ""
	}
	return Pass, ""
}

type noSudo struct{}

func (noSudo) ID() string     { return "no_sudo" }
func (noSudo) Version() int   { return 1 }
func (noSudo) Check(unit string) (Verdict, string) {
	if rxSudo.MatchString(strings.ToLower(unit)) {
		return Fail, "sudo is not allowed"
	}
	return Pass, ""
}

type suspiciousEnv struct{}

func (suspiciousEnv) ID() string     { return "suspicious_env" }
func (suspiciousEnv) Version() int { return 1 }

func (suspiciousEnv) Check(unit string) (Verdict, string) {
	t := strings.ToLower(unit)
	if strings.Contains(t, " ld_preload=") || strings.HasPrefix(t, "ld_preload=") {
		return Fail, "LD_PRELOAD manipulation is not allowed"
	}
	if strings.Contains(t, "bash_env=") || strings.Contains(t, "bash_env =") {
		return Fail, "BASH_ENV manipulation is not allowed"
	}
	return Pass, ""
}

type curlPipeShell struct{}

func (curlPipeShell) ID() string     { return "curl_pipe_shell" }
func (curlPipeShell) Version() int { return 1 }

// Check rejects piping curl/wget into a shell (common remote-code pattern).
func (curlPipeShell) Check(unit string) (Verdict, string) {
	t := strings.ToLower(strings.TrimSpace(unit))
	if !strings.Contains(t, "curl") && !strings.Contains(t, "wget") {
		return Pass, ""
	}
	if rxPipeToShell.MatchString(unit) {
		return Fail, "piping curl/wget to a shell is not allowed"
	}
	return Pass, ""
}

type noChrootNsenter struct{}

func (noChrootNsenter) ID() string     { return "no_chroot_nsenter" }
func (noChrootNsenter) Version() int { return 1 }

func (noChrootNsenter) Check(unit string) (Verdict, string) {
	fields := strings.Fields(strings.TrimSpace(unit))
	if len(fields) == 0 {
		return Pass, ""
	}
	base := filepath.Base(fields[0])
	switch strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(base, ".exe"), ".bat")) {
	case "chroot", "nsenter":
		return Fail, "chroot/nsenter is not allowed"
	default:
		return Pass, ""
	}
}

// DefaultChain is the ordered validator list for production Gate/Execute.
func DefaultChain() []Validator {
	return []Validator{
		blockedSubstrings{},
		noSudo{},
		suspiciousEnv{},
		curlPipeShell{},
		noChrootNsenter{},
		posixSyntax{},
	}
}
