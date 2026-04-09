package tui

import (
	"strings"
	"testing"
	"time"
)

func TestJoinBusyParts(t *testing.T) {
	t.Parallel()
	s := joinBusyParts("a", "", "b")
	if s != "a · b" {
		t.Fatalf("got %q", s)
	}
}

func TestFormatBusyElapsed(t *testing.T) {
	t.Parallel()
	if g := formatBusyElapsed(125 * time.Second); g != "2:05" {
		t.Fatalf("got %q", g)
	}
	if g := formatBusyElapsed(2 * time.Hour); g != "2h00m" {
		t.Fatalf("got %q", g)
	}
}

func TestPickSpinnerVerb(t *testing.T) {
	t.Parallel()
	if len(parsedSpinnerVerbs) == 0 {
		t.Fatal("parsedSpinnerVerbs empty")
	}
	v := pickSpinnerVerb()
	if strings.TrimSpace(v) == "" {
		t.Fatal("empty verb")
	}
}

func TestStripANSI(t *testing.T) {
	t.Parallel()
	raw := stripANSI("\x1b[31mhi\x1b[0m")
	if raw != "hi" {
		t.Fatalf("got %q", raw)
	}
}
