package lifecycle

// Package lifecycle will host ServerConnection state machines (MCP v2 roadmap).

import "github.com/gitlawb/openclaude4/internal/mcp"

// ServerRef is a placeholder for tracked connections.
type ServerRef struct {
	Name  string
	State mcp.ConnectionState
}
