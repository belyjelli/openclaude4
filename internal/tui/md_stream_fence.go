package tui

import "strings"

// isFenceLine reports whether line is a CommonMark-style fenced code delimiter:
// up to 3 leading spaces, then a run of 3+ '`' or '~' (same char), then optional rest.
func isFenceLine(line string) bool {
	line = strings.TrimSuffix(line, "\r")
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return false
	}
	c := line[i]
	if c != '`' && c != '~' {
		return false
	}
	j := i
	for j < len(line) && line[j] == c {
		j++
	}
	return j-i >= 3
}

// splitUnclosedFenceSuffix splits s when the text ends inside an unclosed fenced code block.
// It returns (before, "") when all fences are balanced or there is no fence.
// When unclosed, before is lines before the opening line of the current block; suffix is that
// line through EOF (plain text for streaming).
func splitUnclosedFenceSuffix(s string) (before, suffix string) {
	if s == "" {
		return "", ""
	}
	lines := strings.Split(s, "\n")
	depth := 0
	lastOpenIdx := -1
	for idx, line := range lines {
		if !isFenceLine(line) {
			continue
		}
		if depth == 0 {
			depth = 1
			lastOpenIdx = idx
		} else {
			depth = 0
		}
	}
	if depth == 0 || lastOpenIdx < 0 {
		return s, ""
	}
	start := startByteOfLine(s, lastOpenIdx)
	before = s[:start]
	suffix = s[start:]
	return before, suffix
}

// startByteOfLine returns the byte offset in s where line lineIndex begins (0-based).
func startByteOfLine(s string, lineIndex int) int {
	if lineIndex <= 0 {
		return 0
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
			if n == lineIndex {
				return i + 1
			}
		}
	}
	return len(s)
}
