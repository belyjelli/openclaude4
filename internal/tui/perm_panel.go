package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/gitlawb/openclaude4/internal/toolpolicy"
)

// permTruncatePlain truncates s to fit maxCells display width (single-line).
func permTruncatePlain(s string, maxCells int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if maxCells < 1 {
		return ""
	}
	if ansi.StringWidth(s) <= maxCells {
		return s
	}
	if maxCells == 1 {
		return "…"
	}
	return ansi.Truncate(s, maxCells, "…")
}

// permDetailLines returns human-oriented lines for the permission panel (below "Tool: …").
// maxW is the panel inner width for truncating long commands, paths, and MCP names.
func permDetailLines(tool string, args map[string]any, maxW int) []string {
	tool = strings.TrimSpace(tool)
	var lines []string
	labelBudget := func(labelPlain string) int {
		if maxW < 1 {
			return 400
		}
		w := ansi.StringWidth(labelPlain)
		if w >= maxW {
			return 4
		}
		return maxW - w
	}
	switch {
	case strings.EqualFold(tool, "Bash"):
		cmd := strings.TrimSpace(fmt.Sprint(args["command"]))
		if cmd != "" {
			cmdShow := permTruncatePlain(cmd, labelBudget("Command: "))
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Command")+": "+cmdShow)
		}
		if d := strings.TrimSpace(fmt.Sprint(args["description"])); d != "" {
			lines = append(lines, dimStyle.Render("Note: "+permTruncatePlain(d, labelBudget("Note: "))))
		}
		if h := toolpolicy.BashDestructiveHint(cmd); h != "" {
			lines = append(lines, errStyle.Render(permTruncatePlain(h, max(12, maxW))))
		}
	case strings.EqualFold(tool, "FileWrite"), strings.EqualFold(tool, "FileEdit"):
		p := strings.TrimSpace(fmt.Sprint(args["file_path"]))
		if p != "" {
			pShow := permTruncatePlain(p, labelBudget("Path: "))
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Path")+": "+pShow)
		}
	case strings.EqualFold(tool, "Task"):
		g := strings.TrimSpace(fmt.Sprint(args["goal"]))
		if g != "" {
			gShow := permTruncatePlain(g, labelBudget("Goal: "))
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Goal")+": "+gShow)
		}
	default:
		if strings.HasPrefix(tool, "mcp_") {
			srv, mcpTool := splitMCPToolName(tool)
			lines = append(lines, permTruncatePlain(fmt.Sprintf("MCP server: %s", srv), max(maxW, 8)))
			lines = append(lines, permTruncatePlain(fmt.Sprintf("MCP tool: %s", mcpTool), max(maxW, 8)))
		}
	}
	return lines
}

func splitMCPToolName(openAIName string) (server, tool string) {
	s := strings.TrimPrefix(openAIName, "mcp_")
	i := strings.Index(s, "__")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+2:]
}
