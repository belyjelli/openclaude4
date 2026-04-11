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

func TestRenderAssistantMarkdownChroma_htmlBlockNotDropped(t *testing.T) {
	md := "<details>\n<summary>More</summary>\n\nHidden **bold** here.\n</details>\n"
	out := renderAssistantMarkdownChroma(72, md, true, true)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "details") || !strings.Contains(plain, "Hidden") {
		t.Fatalf("expected HTML block body visible: %q", plain)
	}
}

func TestRenderAssistantMarkdownChroma_inlineRawHTML(t *testing.T) {
	md := "Line one<br>Line two"
	out := renderAssistantMarkdownChroma(72, md, true, true)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Line one") || !strings.Contains(plain, "Line two") {
		t.Fatalf("expected <br> to become line break: %q", plain)
	}
}

func TestRenderAssistantMarkdownChroma_strikethrough(t *testing.T) {
	out := renderAssistantMarkdownChroma(72, "~~gone~~ stays", true, true)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "gone") || !strings.Contains(plain, "stays") {
		t.Fatalf("expected strikethrough segment: %q", plain)
	}
}

// Tight lists: goldmark replaces ListItem Paragraph children with TextBlock (goldmark parser/list.go Close).
// Those must not render as empty in the TUI. Streaming path also parses markdown in model.syncVP;
// incomplete fences use splitUnclosedFenceSuffix — see md_stream_fence_test.go.
func TestRenderAssistantMarkdownChroma_tightListTextBlock(t *testing.T) {
	md := "- **alpha**\n- beta\n"
	out := renderAssistantMarkdownChroma(72, md, true, true)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "alpha") || !strings.Contains(plain, "beta") {
		t.Fatalf("expected tight list item text: %q", plain)
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
