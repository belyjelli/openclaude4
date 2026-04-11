package tui

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// rewriteBracketedPastePaths converts pasted absolute (or resolvable) filesystem paths into @-mentions.
// cwd is the workspace root for shortening (e.g. TUI WorkDir); empty uses current directory for Rel only.
func rewriteBracketedPastePaths(pasted, cwd string) (string, bool) {
	pasted = strings.ReplaceAll(pasted, "\r\n", "\n")
	raw := strings.TrimSpace(pasted)
	if raw == "" {
		return pasted, false
	}

	lines := strings.Split(raw, "\n")
	// Single-line paste (common for drag-one-file)
	if len(lines) == 1 {
		p := cleanPathLine(lines[0])
		if p == "" {
			return pasted, false
		}
		if _, err := os.Stat(p); err != nil {
			return pasted, false
		}
		return atMentionForPath(p, cwd) + " ", true
	}

	var mentions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := cleanPathLine(line)
		if p == "" {
			return pasted, false
		}
		if _, err := os.Stat(p); err != nil {
			return pasted, false
		}
		mentions = append(mentions, atMentionForPath(p, cwd))
	}
	if len(mentions) == 0 {
		return pasted, false
	}
	return strings.Join(mentions, " ") + " ", true
}

func cleanPathLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(s), "file://") {
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		p := u.Path
		if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = strings.TrimPrefix(p, "/")
		}
		return filepath.Clean(p)
	}
	return filepath.Clean(s)
}

func atMentionForPath(abs, cwd string) string {
	rel := abs
	if cwd != "" {
		if awd, err := filepath.Abs(cwd); err == nil {
			if ap, err := filepath.Abs(abs); err == nil {
				if r, err := filepath.Rel(awd, ap); err == nil && !strings.HasPrefix(r, "..") {
					rel = r
				}
			}
		}
	}
	if strings.ContainsAny(rel, " \t") {
		return `@"` + rel + `"`
	}
	return "@" + rel
}
