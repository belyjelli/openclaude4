package tools

import (
	"fmt"
	"strings"
)

// shellSplit splits a simple shell-like command line into words, honoring ' and " quotes.
// Backslash escapes are honored inside double quotes only (matches common gh usage).
func shellSplit(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var fields []string
	var b strings.Builder
	var quote rune
	esc := false
	for _, r := range s {
		if quote == 0 {
			switch r {
			case '\'', '"':
				quote = r
				esc = false
			case ' ', '\t':
				if b.Len() > 0 {
					fields = append(fields, b.String())
					b.Reset()
				}
			default:
				b.WriteRune(r)
			}
			continue
		}
		if quote == '"' {
			if esc {
				b.WriteRune(r)
				esc = false
				continue
			}
			if r == '\\' {
				esc = true
				continue
			}
			if r == '"' {
				quote = 0
				continue
			}
			b.WriteRune(r)
			continue
		}
		// single-quoted: literal until closing '
		if r == '\'' {
			quote = 0
			continue
		}
		b.WriteRune(r)
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in shell command")
	}
	if b.Len() > 0 {
		fields = append(fields, b.String())
	}
	return fields, nil
}

// bashCommandLooksCompound returns true if the string likely needs a real shell
// (subshells, pipes, redirects). We fail closed for read-only gh auto-approval.
func bashCommandLooksCompound(s string) bool {
	if strings.ContainsAny(s, "$`;\n\r|&<>") {
		return true
	}
	if strings.Contains(s, "&&") || strings.Contains(s, "||") {
		return true
	}
	// Two consecutive spaces are fine; look for unquoted # starting a comment is rare — skip.
	return false
}

func isGhExecutable(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	base := tok
	if i := strings.LastIndexAny(tok, `/\`); i >= 0 {
		base = tok[i+1:]
	}
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".exe"), ".bat")
	return base == "gh"
}

func ghTokenLooksLikeExfilHost(value string) bool {
	if value == "" {
		return false
	}
	if !strings.Contains(value, "/") && !strings.Contains(value, "://") && !strings.Contains(value, "@") {
		return false
	}
	if strings.Contains(value, "://") {
		return true
	}
	if strings.Contains(value, "@") {
		return true
	}
	if strings.Count(value, "/") >= 2 {
		return true
	}
	return false
}

// ghArgsHaveSuspiciousRepoOrURL mirrors OpenClaude v3's ghIsDangerousCallback:
// reject HOST/OWNER/REPO (3+ segments), URLs, and ssh-style host specs in flag values and args.
func ghArgsHaveSuspiciousRepoOrURL(args []string) bool {
	for _, token := range args {
		if token == "" {
			continue
		}
		value := token
		if strings.HasPrefix(token, "-") {
			eq := strings.IndexByte(token, '=')
			if eq < 0 {
				continue
			}
			value = token[eq+1:]
			if value == "" {
				continue
			}
		}
		if ghTokenLooksLikeExfilHost(value) {
			return true
		}
	}
	return false
}

