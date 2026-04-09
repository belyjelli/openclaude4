package tui

import (
	"testing"

	"github.com/gitlawb/openclaude4/internal/mcpclient"
)

func TestMcpResourceEntryHint(t *testing.T) {
	tests := []struct {
		r    mcpclient.MCPResource
		want string
	}{
		{mcpclient.MCPResource{Server: "s", Title: "T", Name: "n"}, "s · T"},
		{mcpclient.MCPResource{Server: "s", Name: "only"}, "s · only"},
		{mcpclient.MCPResource{Server: "s"}, "s"},
		{mcpclient.MCPResource{Title: "alone"}, "alone"},
	}
	for _, tc := range tests {
		if got := mcpResourceEntryHint(tc.r); got != tc.want {
			t.Fatalf("%+v: got %q want %q", tc.r, got, tc.want)
		}
	}
}
