package tui

import "testing"

func TestTextInputWidthPositive(t *testing.T) {
	t.Parallel()
	m := &model{
		width:  80,
		height: 24,
	}
	w := m.textInputWidth()
	if w < 1 {
		t.Fatalf("textInputWidth: %d", w)
	}
	if w > 80 {
		t.Fatalf("textInputWidth exceeds terminal: %d", w)
	}
}
