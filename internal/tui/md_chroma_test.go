package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderAssistantMarkdownChroma_headingAndCode(t *testing.T) {
	md := "# Title\n\n```go\nfunc main() {}\n```\n"
	out := renderAssistantMarkdownChroma(72, md, true, true)
	if !strings.Contains(out, "Title") {
		t.Fatalf("expected heading text: %q", out)
	}
	if !strings.Contains(out, "func") {
		t.Fatalf("expected code body: %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI from Chroma: %q", ansi.Strip(out))
	}
}

func TestRenderAssistantMarkdownChroma_table(t *testing.T) {
	md := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	out := renderAssistantMarkdownChroma(72, md, false, true)
	if !strings.Contains(out, "a") || !strings.Contains(out, "|") {
		t.Fatalf("expected table pipe layout: %q", out)
	}
}

func TestRenderAssistantMarkdownChroma_streamingNoTrim(t *testing.T) {
	out := renderAssistantMarkdownChroma(72, "Hello **world**  ", true, false)
	if !strings.Contains(ansi.Strip(out), "Hello") || !strings.Contains(out, "world") {
		t.Fatalf("expected styled output: %q", out)
	}
}

func TestLinkifyIssueRefs(t *testing.T) {
	s := "see org/repo#42 done"
	out := linkifyIssueRefs(s)
	if !strings.Contains(out, "org/repo#42") {
		t.Fatalf("expected label: %q", out)
	}
	if !strings.Contains(out, "\x1b]8;;") && !strings.Contains(out, "https://github.com/org/repo/issues/42") {
		t.Fatalf("expected OSC8 URL: %q", out)
	}
}
