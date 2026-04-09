package main

import (
	"slices"
	"testing"
)

func TestMcpAddCommandArgv(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		bunx bool
		exec []string
		want []string
	}{
		{
			name: "plain",
			exec: []string{"node", "server.js"},
			want: []string{"node", "server.js"},
		},
		{
			name: "bunx",
			bunx: true,
			exec: []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
			want: []string{"bunx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		},
		{
			name: "trim_spaces",
			bunx: true,
			exec: []string{"  pkg  ", "", " arg "},
			want: []string{"bunx", "-y", "pkg", "arg"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mcpAddCommandArgv(tc.bunx, tc.exec)
			if !slices.Equal(tc.want, got) {
				t.Fatalf("mcpAddCommandArgv: want %v, got %v", tc.want, got)
			}
		})
	}
}
