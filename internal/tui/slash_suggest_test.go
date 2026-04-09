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

func TestOverlaySlashOnViewport(t *testing.T) {
	base := strings.Join([]string{"l0", "l1", "l2", "l3", "l4"}, "\n")
	ov := strings.Join([]string{"s0", "s1"}, "\n")
	got := overlaySlashOnViewport(base, ov, 5)
	want := strings.Join([]string{"l0", "l1", "l2", "s0", "s1"}, "\n")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOverlaySlashOnViewportTallOverlayClips(t *testing.T) {
	base := strings.Join([]string{"a", "b", "c"}, "\n")
	ov := strings.Join([]string{"s0", "s1", "s2", "s3"}, "\n")
	got := overlaySlashOnViewport(base, ov, 3)
	want := strings.Join([]string{"s1", "s2", "s3"}, "\n")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
