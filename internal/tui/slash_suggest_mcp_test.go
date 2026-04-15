package tui

import (
	"testing"

	"github.com/gitlawb/openclaude4/internal/mcp"
)

func TestMcpResourceEntryHint(t *testing.T) {
	tests := []struct {
		r    mcp.MCPResource
		want string
	}{
		{mcp.MCPResource{Server: "s", Title: "T", Name: "n"}, "s · T"},
		{mcp.MCPResource{Server: "s", Name: "only"}, "s · only"},
		{mcp.MCPResource{Server: "s"}, "s"},
		{mcp.MCPResource{Title: "alone"}, "alone"},
	}
	for _, tc := range tests {
		if got := mcpResourceEntryHint(tc.r); got != tc.want {
			t.Fatalf("%+v: got %q want %q", tc.r, got, tc.want)
		}
	}
}
