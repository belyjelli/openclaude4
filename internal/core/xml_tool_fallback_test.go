package core

import (
	"strings"
	"testing"
)

func TestStripReasoningBlocks(t *testing.T) {
	s := `<redacted_thinking>secret</redacted_thinking>hi<thinking>t</thinking>there`
	got := stripReasoningBlocks(s)
	if got != "hithere" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractXMLToolCallsFromContent_JSONBody(t *testing.T) {
	s := `ok<tool_call>{"name":"FileRead","arguments":{"file_path":"a.txt"}}</tool_call>`
	calls := extractXMLToolCallsFromContent(s)
	if len(calls) != 1 {
		t.Fatalf("len=%d", len(calls))
	}
	if calls[0].Function.Name != "FileRead" {
		t.Fatalf("name=%q", calls[0].Function.Name)
	}
	if !strings.Contains(calls[0].Function.Arguments, "a.txt") {
		t.Fatalf("args=%q", calls[0].Function.Arguments)
	}
	if calls[0].ID == "" {
		t.Fatal("empty id")
	}
}

func TestExtractXMLToolCallsFromContent_ArgumentsString(t *testing.T) {
	s := `<tool_call>{"name":"Bash","arguments":"{\"command\":\"echo hi\"}"}</tool_call>`
	calls := extractXMLToolCallsFromContent(s)
	if len(calls) != 1 || calls[0].Function.Name != "Bash" {
		t.Fatalf("got %#v", calls)
	}
}

func TestExtractXMLToolCallsFromContent_NameAttribute(t *testing.T) {
	s := `<tool_call name="FileRead">{"file_path":"x.go"}</tool_call>`
	calls := extractXMLToolCallsFromContent(s)
	if len(calls) != 1 {
		t.Fatalf("len=%d", len(calls))
	}
	if calls[0].Function.Name != "FileRead" {
		t.Fatalf("name=%q", calls[0].Function.Name)
	}
	if !strings.Contains(calls[0].Function.Arguments, "x.go") {
		t.Fatalf("args=%q", calls[0].Function.Arguments)
	}
}

func TestExtractXMLToolCallsFromContent_Multiple(t *testing.T) {
	s := `<tool_call>{"name":"A","arguments":{}}</tool_call> <tool_call>{"name":"B","arguments":{}}</tool_call>`
	calls := extractXMLToolCallsFromContent(s)
	if len(calls) != 2 {
		t.Fatalf("len=%d", len(calls))
	}
}

func TestCleanToolCallMarkupFromContent(t *testing.T) {
	s := `intro<tool_call>{}</tool_call> outro`
	got := cleanToolCallMarkupFromContent(s)
	if got != "intro outro" {
		t.Fatalf("got %q", got)
	}
}

func TestXMLToolFallbackEnabledForModel(t *testing.T) {
	t.Setenv(EnvDisableXMLToolFallback, "")
	t.Setenv(EnvXMLToolFallbackAll, "")
	if !xmlToolFallbackEnabledForModel("") {
		t.Fatal("empty model should allow")
	}
	if !xmlToolFallbackEnabledForModel("Qwen3-35B") {
		t.Fatal("qwen should allow")
	}
	if xmlToolFallbackEnabledForModel("gpt-4o-mini") {
		t.Fatal("non-qwen should deny without all")
	}
	t.Setenv(EnvXMLToolFallbackAll, "all")
	if !xmlToolFallbackEnabledForModel("gpt-4o-mini") {
		t.Fatal("all should force")
	}
	t.Setenv(EnvXMLToolFallbackAll, "")
	t.Setenv(EnvDisableXMLToolFallback, "1")
	if xmlToolFallbackEnabledForModel("qwen") {
		t.Fatal("disable should win")
	}
}
