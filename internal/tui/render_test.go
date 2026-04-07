package tui

import "testing"

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
	got := formatToolResultBody(4, s, 80)
	if got != "a..." {
		t.Fatalf("got %q", got)
	}
}
