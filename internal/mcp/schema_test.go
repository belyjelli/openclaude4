package mcp

import "testing"

func TestSanitizeOpenAIName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "x"},
		{"my-server", "my-server"},
		{"a b", "a_b"},
		{"123", "n_123"},
	}
	for _, tt := range tests {
		if got := SanitizeOpenAIName(tt.in); got != tt.want {
			t.Errorf("SanitizeOpenAIName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOpenAIToolName(t *testing.T) {
	got := OpenAIToolName("fs", "read_file")
	if got != "mcp_fs__read_file" {
		t.Fatalf("got %q", got)
	}
}

func TestInputSchemaToParameters(t *testing.T) {
	m := InputSchemaToParameters(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
		},
	})
	if m["type"] != "object" {
		t.Fatalf("type = %v", m["type"])
	}
	props, _ := m["properties"].(map[string]any)
	if props == nil || props["path"] == nil {
		t.Fatalf("properties: %v", m["properties"])
	}
}

func TestUniqueOpenAIName(t *testing.T) {
	used := map[string]struct{}{"mcp_a__b": {}}
	got := UniqueOpenAIName("mcp_a__b", used)
	if got == "mcp_a__b" {
		t.Fatalf("expected suffix, got %q", got)
	}
	if _, ok := used[got]; ok {
		t.Fatalf("returned name should not be in used yet: %q", got)
	}
}
