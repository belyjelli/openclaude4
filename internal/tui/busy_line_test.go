package tui

import (
	"strings"
	"testing"
	"time"
)

func TestStallTargetIntensityToolSuppresses(t *testing.T) {
	t.Parallel()
	m := &model{
		runningTool:      "bash",
		busyStart:        time.Now().Add(-10 * time.Minute),
		lastStreamChange: time.Now().Add(-10 * time.Minute),
	}
	if m.stallTargetIntensity(time.Now()) != 0 {
		t.Fatalf("expected 0 with active tool")
	}
}

func TestStallTargetIntensitySchedulingSuppresses(t *testing.T) {
	t.Parallel()
	m := &model{
		pendingToolScheduleCount: 2,
		busyStart:                time.Now().Add(-10 * time.Minute),
		lastStreamChange:         time.Now().Add(-10 * time.Minute),
	}
	if m.stallTargetIntensity(time.Now()) != 0 {
		t.Fatalf("expected 0 while scheduling tools before first KindToolCall")
	}
}

func TestPickBusyLineVerbOverride(t *testing.T) {
	t.Parallel()
	m := &model{cfg: Config{
		BusySpinnerVerb: func() string { return "  CustomTask  " },
	}}
	if g := m.pickBusyLineVerb(); g != "CustomTask" {
		t.Fatalf("got %q", g)
	}
}

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

func TestPickBusyLineVerb(t *testing.T) {
	t.Parallel()
	if len(parsedSpinnerVerbs) == 0 {
		t.Fatal("parsedSpinnerVerbs empty")
	}
	m := &model{cfg: Config{}}
	v := m.pickBusyLineVerb()
	if strings.TrimSpace(v) == "" {
		t.Fatal("empty verb")
	}
}

func TestFormatBusyInt(t *testing.T) {
	t.Parallel()
	if g := formatBusyInt(999); g != "999" {
		t.Fatalf("got %q", g)
	}
	if g := formatBusyInt(10000); g != "10,000" {
		t.Fatalf("got %q", g)
	}
}

func TestStripANSI(t *testing.T) {
	t.Parallel()
	raw := stripANSI("\x1b[31mhi\x1b[0m")
	if raw != "hi" {
		t.Fatalf("got %q", raw)
	}
}
