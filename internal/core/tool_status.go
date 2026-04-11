package core

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const toolCallBusyLabelMaxRunes = 72

// ToolCallBusyLabel returns a short, human-oriented one-line summary for the TUI busy strip.
// toolName is the OpenAI function name; args should already be redacted when sourced from Event.ToolArgs.
func ToolCallBusyLabel(toolName string, args map[string]any) string {
	s := toolCallBusyLabelInner(toolName, args)
	s = strings.TrimSpace(RedactStringForLog(s))
	if s == "" {
		return toolName
	}
	if toolName != "" && !strings.EqualFold(s, toolName) {
		return toolName + ": " + s
	}
	return s
}

func toolCallBusyLabelInner(toolName string, args map[string]any) string {
	if args == nil {
		return toolName
	}
	switch toolName {
	case "Bash":
		if cmd, ok := stringArg(args["command"]); ok {
			return oneLineSnippet(cmd, toolCallBusyLabelMaxRunes-6)
		}
	case "FileRead", "FileWrite", "FileEdit":
		if p, ok := stringArg(args["file_path"]); ok {
			return oneLineSnippet(p, toolCallBusyLabelMaxRunes-6)
		}
	case "Grep":
		var parts []string
		if pat, ok := stringArg(args["pattern"]); ok {
			parts = append(parts, oneLineSnippet(pat, 36))
		}
		if path, ok := stringArg(args["path"]); ok && path != "" && path != "." {
			parts = append(parts, oneLineSnippet(path, 28))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	case "Glob":
		if pat, ok := stringArg(args["pattern"]); ok {
			return oneLineSnippet(pat, toolCallBusyLabelMaxRunes-6)
		}
	case "WebSearch":
		if q, ok := stringArg(args["query"]); ok {
			return oneLineSnippet(q, toolCallBusyLabelMaxRunes-10)
		}
	case "WebFetch":
		if u, ok := stringArg(args["url"]); ok {
			return oneLineSnippet(u, toolCallBusyLabelMaxRunes-6)
		}
	}

	for _, k := range []string{"command", "file_path", "path", "pattern", "query", "url", "glob", "target_file"} {
		if s, ok := stringArg(args[k]); ok && strings.TrimSpace(s) != "" {
			return oneLineSnippet(s, toolCallBusyLabelMaxRunes-6)
		}
	}

	// Fallback: compact JSON, capped (args expected small for typical tools).
	b, err := json.Marshal(args)
	if err != nil || len(b) == 0 {
		return toolName
	}
	s := string(b)
	return oneLineSnippet(s, toolCallBusyLabelMaxRunes)
}

func stringArg(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func oneLineSnippet(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(s), " ")
	if maxRunes < 8 {
		maxRunes = 8
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes-1 {
			b.WriteRune('…')
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
