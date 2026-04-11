package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gitlawb/openclaude4/internal/toolpolicy"
)

// permDetailLines returns human-oriented lines for the permission panel (below "Tool: …").
func permDetailLines(tool string, args map[string]any) []string {
	tool = strings.TrimSpace(tool)
	var lines []string
	switch {
	case strings.EqualFold(tool, "Bash"):
		cmd := strings.TrimSpace(fmt.Sprint(args["command"]))
		if cmd != "" {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Command")+": "+cmd)
		}
		if d := strings.TrimSpace(fmt.Sprint(args["description"])); d != "" {
			lines = append(lines, dimStyle.Render("Note: "+d))
		}
		if h := toolpolicy.BashDestructiveHint(cmd); h != "" {
			lines = append(lines, errStyle.Render(h))
		}
	case strings.EqualFold(tool, "FileWrite"), strings.EqualFold(tool, "FileEdit"):
		p := strings.TrimSpace(fmt.Sprint(args["file_path"]))
		if p != "" {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Path")+": "+p)
		}
	case strings.EqualFold(tool, "Task"):
		g := strings.TrimSpace(fmt.Sprint(args["goal"]))
		if g != "" {
			if len(g) > 200 {
				g = g[:197] + "..."
			}
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Goal")+": "+g)
		}
	default:
		if strings.HasPrefix(tool, "mcp_") {
			srv, mcpTool := splitMCPToolName(tool)
			lines = append(lines, fmt.Sprintf("MCP server: %s", srv))
			lines = append(lines, fmt.Sprintf("MCP tool: %s", mcpTool))
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
