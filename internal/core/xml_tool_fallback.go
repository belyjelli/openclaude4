package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

// EnvDisableXMLToolFallback disables parsing <tool_call> XML when the stream has no delta.ToolCalls.
const EnvDisableXMLToolFallback = "OPENCLAUDE_DISABLE_XML_TOOL_FALLBACK"

// EnvXMLToolFallbackAll enables XML tool fallback for every model (not only names containing "qwen").
const EnvXMLToolFallbackAll = "OPENCLAUDE_XML_TOOL_FALLBACK"

var (
	reRedactedThinking = regexp.MustCompile(`(?i)<redacted_thinking>[\s\S]*?</redacted_thinking>`)
	reThinkingBlock    = regexp.MustCompile(`(?i)<thinking>[\s\S]*?</thinking>`)
)

func stripReasoningBlocks(s string) string {
	s = reRedactedThinking.ReplaceAllString(s, "")
	s = reThinkingBlock.ReplaceAllString(s, "")
	return s
}

// cleanToolCallMarkupFromContent removes well-formed <tool_call>...</tool_call> blocks (case-insensitive).
func cleanToolCallMarkupFromContent(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(lower[i:], "<tool_call")
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		j += i
		b.WriteString(s[i:j])
		gt := strings.Index(s[j:], ">")
		if gt < 0 {
			b.WriteString(s[j:])
			break
		}
		openEnd := j + gt + 1
		cl := strings.Index(lower[openEnd:], "</tool_call>")
		if cl < 0 {
			b.WriteString(s[j:])
			break
		}
		i = openEnd + cl + len("</tool_call>")
	}
	return strings.TrimSpace(b.String())
}

// xmlToolFallbackEnabledForModel returns whether we may run XML tool-call extraction after the stream.
// Disabled when OPENCLAUDE_DISABLE_XML_TOOL_FALLBACK=1. Forced on for all models when OPENCLAUDE_XML_TOOL_FALLBACK=all.
// Otherwise: empty model id allows fallback (local GGUF names); non-empty requires "qwen" in the model string.
func xmlToolFallbackEnabledForModel(model string) bool {
	if os.Getenv(EnvDisableXMLToolFallback) == "1" {
		return false
	}
	if os.Getenv(EnvXMLToolFallbackAll) == "all" {
		return true
	}
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return true
	}
	return strings.Contains(m, "qwen")
}

func newXMLFallbackToolCallID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "call_xml_fallback"
	}
	return "call_xml_" + hex.EncodeToString(buf[:])
}

func parseToolCallNameAttr(openTag string) string {
	lo := strings.ToLower(openTag)
	idx := strings.Index(lo, "name=")
	if idx < 0 {
		return ""
	}
	rest := openTag[idx+len("name="):]
	if len(rest) == 0 {
		return ""
	}
	q := rest[0]
	if q != '"' && q != '\'' {
		return ""
	}
	end := strings.IndexByte(rest[1:], q)
	if end < 0 {
		return ""
	}
	return rest[1 : 1+end]
}

// balanceJSONObject returns the substring from the first '{' through the matching '}' using string-aware brace depth.
func balanceJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 || s[0] != '{' {
		return s
	}
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return s
}

func normalizeArgumentsField(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case map[string]any:
		b, err := json.Marshal(x)
		if err != nil {
			return "", false
		}
		return string(b), true
	case nil:
		return "{}", true
	default:
		return "", false
	}
}

func interpretToolCallInner(attrName, inner string) (name, args string) {
	inner = strings.TrimSpace(inner)
	if attrName != "" {
		if inner == "" {
			return attrName, "{}"
		}
		return attrName, inner
	}
	if inner == "" || inner[0] != '{' {
		return "", ""
	}
	jsonStr := balanceJSONObject(inner)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return "", ""
	}
	n, _ := parsed["name"].(string)
	if n == "" {
		return "", ""
	}
	if a, ok := parsed["arguments"]; ok {
		s, ok := normalizeArgumentsField(a)
		if !ok {
			return "", ""
		}
		return n, s
	}
	if p, ok := parsed["parameters"]; ok {
		s, ok := normalizeArgumentsField(p)
		if !ok {
			return "", ""
		}
		return n, s
	}
	return n, "{}"
}

// extractXMLToolCallsFromContent parses Qwen-style <tool_call> blocks from assistant text when OpenAI delta.tool_calls is empty.
func extractXMLToolCallsFromContent(s string) []sdk.ToolCall {
	var out []sdk.ToolCall
	lower := strings.ToLower(s)
	i := 0
	for {
		j := strings.Index(lower[i:], "<tool_call")
		if j < 0 {
			break
		}
		j += i
		gt := strings.Index(s[j:], ">")
		if gt < 0 {
			break
		}
		openEnd := j + gt + 1
		openTag := s[j:openEnd]
		cl := strings.Index(lower[openEnd:], "</tool_call>")
		if cl < 0 {
			break
		}
		inner := strings.TrimSpace(s[openEnd : openEnd+cl])
		attrName := parseToolCallNameAttr(openTag)
		i = openEnd + cl + len("</tool_call>")

		name, args := interpretToolCallInner(attrName, inner)
		if name == "" {
			continue
		}
		out = append(out, sdk.ToolCall{
			ID:   newXMLFallbackToolCallID(),
			Type: sdk.ToolTypeFunction,
			Function: sdk.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		})
	}
	return out
}
