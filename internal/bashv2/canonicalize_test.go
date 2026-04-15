package bashv2

import "testing"

func TestCanonicalizeCommand(t *testing.T) {
	t.Parallel()
	in := "# intro\nls\n# tail"
	got := CanonicalizeCommand(in)
	if got != "ls" {
		t.Fatalf("got %q", got)
	}
	all := "# only\n# lines"
	if CanonicalizeCommand(all) != all {
		t.Fatal("all-comment input should be unchanged")
	}
}
