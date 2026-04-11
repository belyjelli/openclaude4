package tools

import "testing"

func TestStripBashCommentLines(t *testing.T) {
	t.Parallel()
	in := "# intro\nls\n# tail"
	got := stripBashCommentLines(in)
	if got != "ls" {
		t.Fatalf("got %q", got)
	}
	all := "# only\n# lines"
	if stripBashCommentLines(all) != all {
		t.Fatal("all-comment input should be unchanged")
	}
}
