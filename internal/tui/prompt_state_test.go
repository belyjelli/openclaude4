package tui

import "testing"

func TestHistoryIndicesMatchingPrefix(t *testing.T) {
	h := []string{"alpha", "git log", "other", "git status"}
	got := historyIndicesMatchingPrefix(h, "git")
	// newest first: git status (3), git log (1)
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("got %v", got)
	}
	if len(historyIndicesMatchingPrefix(h, "nomatch")) != 0 {
		t.Fatal("expected empty")
	}
	if historyIndicesMatchingPrefix(h, "") != nil {
		t.Fatal("empty prefix should return nil")
	}
	idx := historyIndicesMatchingPrefix(h, "  GIT  ")
	if len(idx) != 2 || idx[0] != 3 || idx[1] != 1 {
		t.Fatalf("case/trim: got %v", idx)
	}
}
