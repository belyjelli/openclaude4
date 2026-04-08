package tui

import (
	"slices"
	"strings"
	"testing"
)

func TestFilterSlashEntriesPrefix(t *testing.T) {
	all := buildSlashIndex(nil)
	got := filterSlashEntries(all, "m")
	var primaries []string
	for _, e := range got {
		primaries = append(primaries, e.primary)
	}
	for _, want := range []string{"mcp", "model"} {
		if !slices.Contains(primaries, want) {
			t.Fatalf("expected %q in matches, got %v", want, primaries)
		}
	}
}

func TestFilterSlashEntriesHelp(t *testing.T) {
	all := buildSlashIndex(nil)
	got := filterSlashEntries(all, "help")
	if len(got) != 1 || got[0].primary != "help" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterSlashEntriesEmptyStem(t *testing.T) {
	all := buildSlashIndex(nil)
	got := filterSlashEntries(all, "")
	if len(got) != len(all) {
		t.Fatalf("empty stem should match all, got %d want %d", len(got), len(all))
	}
}

func TestFilterSlashEntriesCaseFold(t *testing.T) {
	all := buildSlashIndex(nil)
	lo := filterSlashEntries(all, "mo")
	up := filterSlashEntries(all, "MO")
	if len(lo) == 0 || len(lo) != len(up) {
		t.Fatalf("lo=%d up=%d", len(lo), len(up))
	}
}

func TestBuildSlashIndexSkills(t *testing.T) {
	names := func() []string { return []string{"AlphaSkill", "beta"} }
	all := buildSlashIndex(names)
	got := filterSlashEntries(all, "alpha")
	if len(got) != 1 || got[0].primary != "AlphaSkill" {
		t.Fatalf("got %#v", got)
	}
	// Duplicate static name skipped
	all2 := buildSlashIndex(func() []string { return []string{"help", "model"} })
	for _, e := range all2 {
		if e.primary == "help" && e.hint == "skill" {
			t.Fatal("should not add duplicate help as skill")
		}
	}
}

func TestVisibleSlashWindow(t *testing.T) {
	matches := make([]slashEntry, 10)
	for i := range matches {
		matches[i] = slashEntry{primary: strings.Repeat("x", i) + "c"}
	}
	start, win := visibleSlashWindow(matches, 5)
	if len(win) != slashSuggestMaxRows {
		t.Fatalf("len(win)=%d", len(win))
	}
	if start < 0 || start+len(win) > len(matches) {
		t.Fatalf("invalid window start=%d len=%d", start, len(win))
	}
	if start > 5 || start+len(win) <= 5 {
		t.Fatalf("expected window to include index 5, start=%d", start)
	}
}

func TestSuggestionBlockHeightMoreRow(t *testing.T) {
	few := []slashEntry{{primary: "a"}, {primary: "b"}}
	if suggestionBlockHeight(few) != slashSuggestHeaderLines+2 {
		t.Fatalf("few: %d", suggestionBlockHeight(few))
	}
	many := make([]slashEntry, slashSuggestMaxRows+3)
	if h := suggestionBlockHeight(many); h != slashSuggestHeaderLines+slashSuggestMaxRows+1 {
		t.Fatalf("many: %d", h)
	}
}
