package core

import (
	"strings"
	"testing"
)

func TestToolCallBusyLabel_Bash(t *testing.T) {
	t.Parallel()
	s := ToolCallBusyLabel("Bash", map[string]any{"command": "ls -la"})
	if !strings.Contains(s, "Bash") || !strings.Contains(s, "ls") {
		t.Fatalf("got %q", s)
	}
}

func TestToolCallBusyLabel_FileRead(t *testing.T) {
	t.Parallel()
	s := ToolCallBusyLabel("FileRead", map[string]any{"file_path": "internal/foo.go"})
	if !strings.Contains(s, "FileRead") || !strings.Contains(s, "foo.go") {
		t.Fatalf("got %q", s)
	}
}

func TestToolCallBusyLabel_Grep(t *testing.T) {
	t.Parallel()
	s := ToolCallBusyLabel("Grep", map[string]any{"pattern": "func main", "path": "cmd"})
	if !strings.Contains(s, "Grep") || !strings.Contains(s, "main") {
		t.Fatalf("got %q", s)
	}
}

func TestToolCallBusyLabel_LongCommandTruncates(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 200)
	s := ToolCallBusyLabel("Bash", map[string]any{"command": long})
	if utf8RuneCount(s) > toolCallBusyLabelMaxRunes+20 {
		t.Fatalf("too long: %d runes: %q", utf8RuneCount(s), s)
	}
	if !strings.Contains(s, "…") {
		t.Fatalf("expected ellipsis in %q", s)
	}
}

func TestToolCallBusyLabel_RedactsBearer(t *testing.T) {
	t.Parallel()
	s := ToolCallBusyLabel("Bash", map[string]any{"command": `curl -H "Authorization: Bearer sk-secret-token-here" https://x`})
	if strings.Contains(s, "sk-secret") {
		t.Fatalf("secret leaked: %q", s)
	}
	if !strings.Contains(s, RedactedPlaceholder) {
		t.Fatalf("expected redaction in %q", s)
	}
}

func utf8RuneCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
