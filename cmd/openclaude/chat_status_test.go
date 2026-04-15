package main

import (
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/mcp"
)

func TestTuiMCPStatusFragment(t *testing.T) {
	if s := tuiMCPStatusFragment(nil); s != "" {
		t.Fatalf("nil: %q", s)
	}
	if s := tuiMCPStatusFragment(&mcp.Manager{}); s != "" {
		t.Fatalf("empty: %q", s)
	}
	m := &mcp.Manager{Servers: []mcp.ServerTools{
		{OpenAINames: []string{"a"}},
		{OpenAINames: []string{"b", "c"}},
	}}
	got := tuiMCPStatusFragment(m)
	if !strings.Contains(got, "3") || !strings.Contains(got, "2 srv") {
		t.Fatalf("got %q", got)
	}
}
