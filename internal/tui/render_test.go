package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLooksLikeDiff(t *testing.T) {
	t.Parallel()
	if looksLikeDiff("a\nb\nc") {
		t.Fatal("plain text should not look like diff")
	}
	unified := `--- a/foo
+++ b/foo
@@ -1,2 +1,2 @@
 line
-old
+new
`
	if !looksLikeDiff(unified) {
		t.Fatal("expected unified diff")
	}
}

func TestFormatToolResultBodyTruncate(t *testing.T) {
	t.Parallel()
	s := string([]rune{'a', 'b', 'c', 'd', 'e'})
	got := formatToolResultBody(4, 0, s, 80)
	if got != "a..." {
		t.Fatalf("got %q", got)
	}
}

func TestFormatToolResultBodyMaxLines(t *testing.T) {
	t.Parallel()
	s := "one\ntwo\nthree\nfour"
	got := formatToolResultBody(100, 2, s, 80)
	if !strings.HasPrefix(got, "one\ntwo") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(ansi.Strip(got), "more line") {
		t.Fatalf("expected omission hint: %q", got)
	}
}

// renderAssistantMarkdown is the finished-turn path (model KindAssistantFinished).
// When disabled (e.g. OPENCLAUDE_TUI_MARKDOWN=0 → MarkdownAssist false), callers fall back to plain lipgloss body.
func TestRenderAssistantMarkdown_disabled(t *testing.T) {
	t.Parallel()
	if got := renderAssistantMarkdown(80, "# Title", false, "dark"); got != "" {
		t.Fatalf("disabled: want empty, got %q", got)
	}
}

func TestRenderAssistantMarkdown_enabled(t *testing.T) {
	t.Parallel()
	out := renderAssistantMarkdown(80, "# Title", true, "dark")
	if !strings.Contains(ansi.Strip(out), "Title") {
		t.Fatalf("enabled: expected heading text: %q", ansi.Strip(out))
	}
}

func TestRenderAssistantMarkdown_whitespaceOnly(t *testing.T) {
	t.Parallel()
	if got := renderAssistantMarkdown(80, "  \n\t  ", true, "dark"); got != "" {
		t.Fatalf("whitespace-only after trim: want empty, got %q", got)
	}
}
