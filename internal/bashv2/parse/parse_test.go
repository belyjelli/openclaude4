package parse

import "testing"

func TestSplitUnits(t *testing.T) {
	t.Parallel()
	u, err := SplitUnits(`echo a && echo b`)
	if err != nil {
		t.Fatal(err)
	}
	if len(u) != 2 {
		t.Fatalf("got %d units: %v", len(u), u)
	}
}

func TestSplitUnits_respectsQuotes(t *testing.T) {
	t.Parallel()
	u, err := SplitUnits(`echo "a;b" && echo c`)
	if err != nil {
		t.Fatal(err)
	}
	if len(u) != 2 {
		t.Fatalf("got %d units: %v", len(u), u)
	}
}
