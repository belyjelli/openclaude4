package ghstatus

import "testing"

func TestOpenPRStatus_FormatShort(t *testing.T) {
	s := &OpenPRStatus{Number: 42, ReviewState: PrPending}
	if g := s.FormatShort(); g != "PR #42 · pending" {
		t.Fatalf("got %q", g)
	}
}
