package mcp

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var nonOpenAIName = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// SanitizeOpenAIName maps a segment to a safe OpenAI function-name fragment.
func SanitizeOpenAIName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	s = nonOpenAIName.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "x"
	}
	runes := []rune(s)
	if unicode.IsDigit(runes[0]) {
		s = "n_" + s
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

// OpenAIToolName is the function name exposed to the model for an MCP tool.
func OpenAIToolName(serverName, mcpToolName string) string {
	a := SanitizeOpenAIName(serverName)
	b := SanitizeOpenAIName(mcpToolName)
	return "mcp_" + a + "__" + b
}

// InputSchemaToParameters converts an MCP inputSchema value into a JSON-schema-style map for OpenAI tools.
func InputSchemaToParameters(schema any) map[string]any {
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	var m map[string]any
	switch x := schema.(type) {
	case map[string]any:
		m = cloneStringAnyMap(x)
	default:
		b, err := json.Marshal(schema)
		if err != nil {
			return map[string]any{"type": "object", "properties": map[string]any{}}
		}
		if err := json.Unmarshal(b, &m); err != nil || m == nil {
			return map[string]any{"type": "object", "properties": map[string]any{}}
		}
	}
	if _, ok := m["type"]; !ok {
		m["type"] = "object"
	}
	if _, ok := m["properties"]; !ok {
		m["properties"] = map[string]any{}
	}
	return m
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// UniqueOpenAIName returns name if unused; otherwise appends _2, _3, … (kept within a reasonable length).
func UniqueOpenAIName(name string, used map[string]struct{}) string {
	if used == nil {
		return name
	}
	base := name
	for i := 0; i < 1000; i++ {
		var candidate string
		if i == 0 {
			candidate = base
		} else {
			suffix := "_" + strconv.Itoa(i+1)
			if len(base)+len(suffix) > 64 {
				trim := 64 - len(suffix)
				if trim < 1 {
					trim = 1
				}
				candidate = base[:trim] + suffix
			} else {
				candidate = base + suffix
			}
		}
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
	return base + "_dup"
}
