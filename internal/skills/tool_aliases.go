package skills

import "strings"

// ResolveCanonicalToolName maps common v3 / Claude-style tool labels to v4 registry names.
func ResolveCanonicalToolName(s string) string {
	key := strings.TrimSpace(s)
	if key == "" {
		return ""
	}
	aliases := map[string]string{
		"Read":         "FileRead",
		"Write":        "FileWrite",
		"Edit":         "FileEdit",
		"Bash":         "Bash",
		"Grep":         "Grep",
		"Glob":         "Glob",
		"WebSearch":    "WebSearch",
		"WebFetch":     "WebFetch",
		"Task":         "Task",
		"SkillsList":   "SkillsList",
		"SkillsRead":   "SkillsRead",
		"GoOutline":    "GoOutline",
		"SpiderScrape": "SpiderScrape",
		"PaperCLI":     "PaperCLI",
		"SpeedtestCLI": "SpeedtestCLI",
	}
	if v, ok := aliases[key]; ok {
		return v
	}
	// MCP tools: mcp_server__tool style — pass through
	if strings.HasPrefix(key, "mcp_") {
		return key
	}
	return key
}

// NormalizeAllowedToolList resolves aliases and drops empties.
func NormalizeAllowedToolList(in []string) []string {
	var out []string
	for _, s := range in {
		n := ResolveCanonicalToolName(s)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}
