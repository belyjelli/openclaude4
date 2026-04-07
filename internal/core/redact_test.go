package core

import (
	"strings"
	"testing"
)

func TestRedactStringForLog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "bearer",
			in:   `curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.x.y" https://x`,
			want: `curl -H "Authorization: [REDACTED] https://x`,
		},
		{
			name: "authorization header line",
			in:   "Authorization: Basic dGVzdDp0ZXN0\nnext",
			want: "Authorization: [REDACTED]\nnext",
		},
		{
			name: "env api key",
			in:   "export OPENAI_API_KEY=sk-123456789012345678901234",
			want: "export OPENAI_API_KEY=[REDACTED]",
		},
		{
			name: "openai sk key",
			in:   "key is sk-abcdefghijklmnopqrstuvwxyz0123456789",
			want: "key is [REDACTED]",
		},
		{
			name: "google style",
			in:   "k=AIzaSyDxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			want: "k=[REDACTED]",
		},
		{
			name: "json api_key field",
			in:   `{"api_key":"supersecret123","ok":true}`,
			want: `{"api_key":"[REDACTED]","ok":true}`,
		},
		{
			name: "long base64 blob",
			in:   "x " + strings.Repeat("A", 80) + " y",
			want: "x [REDACTED] y",
		},
		{
			name: "short alphanumeric unchanged",
			in:   "abcdABCD1234",
			want: "abcdABCD1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RedactStringForLog(tt.in)
			if got != tt.want {
				t.Fatalf("RedactStringForLog(...) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactEventForLog_toolArgsCopy(t *testing.T) {
	t.Parallel()
	orig := map[string]any{
		"api_key": "secret123",
		"file":    "README.md",
	}
	ev := Event{
		Kind:         KindToolCall,
		UserText:     "token sk-abcdefghijklmnopqrstuvwxyz0123456789 please",
		ToolArgs:     orig,
		ToolArgsJSON: `{"api_key":"x"}`,
	}
	red := RedactEventForLog(ev)
	if got := orig["api_key"]; got != "secret123" {
		t.Fatalf("original map mutated: api_key=%v", got)
	}
	if red.ToolArgs["api_key"] != RedactedPlaceholder {
		t.Fatalf("redacted ToolArgs api_key = %v", red.ToolArgs["api_key"])
	}
	if red.ToolArgs["file"] != "README.md" {
		t.Fatalf("file = %v", red.ToolArgs["file"])
	}
	if !strings.Contains(red.UserText, RedactedPlaceholder) {
		t.Fatalf("UserText not redacted: %q", red.UserText)
	}
	if !strings.Contains(red.ToolArgsJSON, RedactedPlaceholder) {
		t.Fatalf("ToolArgsJSON not redacted: %q", red.ToolArgsJSON)
	}
}

func TestFormatToolArgsForLog(t *testing.T) {
	t.Parallel()
	s := FormatToolArgsForLog(map[string]any{
		"command": "echo hi",
		"token":   "hunter2",
	})
	if strings.Contains(s, "hunter2") {
		t.Fatalf("expected token redacted, got %s", s)
	}
	if !strings.Contains(s, RedactedPlaceholder) {
		t.Fatalf("expected placeholder in %s", s)
	}
}
