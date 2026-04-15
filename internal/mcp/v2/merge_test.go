package v2

import "testing"

func TestMergeServerLayers_lastWins(t *testing.T) {
	a := []Server{{Name: "x", Command: []string{"a"}}}
	b := []Server{{Name: "x", Command: []string{"b"}}}
	got := MergeServerLayers(a, b)
	if len(got) != 1 || got[0].Command[0] != "b" {
		t.Fatalf("got %#v", got)
	}
}

func TestMergeServerLayers_distinctNames(t *testing.T) {
	a := []Server{{Name: "a", Command: []string{"1"}}}
	b := []Server{{Name: "b", Command: []string{"2"}}}
	got := MergeServerLayers(a, b)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}
