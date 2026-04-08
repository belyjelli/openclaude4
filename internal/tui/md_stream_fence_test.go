package tui

import (
	"strings"
	"testing"
)

func TestIsFenceLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"```", true},
		{"````", true},
		{"```go", true},
		{"  ```", true},
		{"   ```", true},
		{"    ```", false},
		{"~~~", true},
		{"  ~~~bash", true},
		{"text", false},
		{"``", false},
		{"`not a fence`", false},
		{"\r", false},
		{"```\r", true},
	}
	for _, tt := range tests {
		if got := isFenceLine(tt.line); got != tt.want {
			t.Errorf("isFenceLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestSplitUnclosedFenceSuffix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		before string
		suffix string
	}{
		{
			name:   "empty",
			input:  "",
			before: "",
			suffix: "",
		},
		{
			name:   "no fence",
			input:  "Hello **world**",
			before: "Hello **world**",
			suffix: "",
		},
		{
			name:   "closed fence",
			input:  "```go\nx\n```\n",
			before: "```go\nx\n```\n",
			suffix: "",
		},
		{
			name:   "open fence eof",
			input:  "Intro\n\n```go\npartial",
			before: "Intro\n\n",
			suffix: "```go\npartial",
		},
		{
			name:   "only open fence",
			input:  "```python\nprint(1)",
			before: "",
			suffix: "```python\nprint(1)",
		},
		{
			name:   "tilde fence open",
			input:  "x\n~~~\ncode",
			before: "x\n",
			suffix: "~~~\ncode",
		},
		{
			name:   "indented fence open",
			input:  "p\n  ```\nline",
			before: "p\n",
			suffix: "  ```\nline",
		},
		{
			name:   "two blocks second open",
			input:  "```\na\n```\n\n```\nb",
			before: "```\na\n```\n\n",
			suffix: "```\nb",
		},
		{
			name:   "whitespace before first fence",
			input:  "\n\n```\nx",
			before: "\n\n",
			suffix: "```\nx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, suf := splitUnclosedFenceSuffix(tt.input)
			if b != tt.before {
				t.Errorf("before = %q, want %q", b, tt.before)
			}
			if suf != tt.suffix {
				t.Errorf("suffix = %q, want %q", suf, tt.suffix)
			}
			if suf == "" && b != tt.input {
				t.Errorf("empty suffix: before should equal input, got %q vs %q", b, tt.input)
			}
		})
	}
}

func TestSplitUnclosedFenceSuffix_crlf(t *testing.T) {
	input := "hi\r\n```\r\npartial"
	b, suf := splitUnclosedFenceSuffix(input)
	if b != "hi\r\n" || !strings.HasPrefix(suf, "```") || !strings.Contains(suf, "partial") {
		t.Fatalf("before=%q suffix=%q", b, suf)
	}
}
