package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/gitlawb/openclaude4/internal/core"
)

func TestApplyKernel_AssistantFinishedToolsOnly_PlaceholderAndScheduling(t *testing.T) {
	t.Parallel()
	m := &model{
		width:    100,
		busy:     true,
		busyVerb: "Testing",
	}
	m.applyKernel(core.Event{
		Kind:          core.KindAssistantFinished,
		ToolCallCount: 2,
		AssistantText: "",
	})
	if m.pendingToolScheduleCount != 2 {
		t.Fatalf("pendingToolScheduleCount=%d", m.pendingToolScheduleCount)
	}
	s := m.committed.String()
	if !strings.Contains(s, "Assistant") || !strings.Contains(s, "2 tool calls") {
		t.Fatalf("expected placeholder in transcript, got: %q", s)
	}
	if m.headerSubLines() < 2 {
		t.Fatalf("headerSubLines=%d want >=2 when busy", m.headerSubLines())
	}
	line := m.renderBusyAnimationLine()
	if !strings.Contains(stripANSI(line), "scheduling 2 tool calls") {
		t.Fatalf("busy line missing scheduling: %q", line)
	}
}

func TestApplyKernel_ToolCallClearsSchedulingAndShowsSummary(t *testing.T) {
	t.Parallel()
	m := &model{
		width:                    100,
		busy:                     true,
		busyVerb:                 "Working",
		pendingToolScheduleCount: 2,
	}
	m.applyKernel(core.Event{
		Kind:     core.KindToolCall,
		ToolName: "Bash",
		ToolArgs: map[string]any{"command": "echo hi"},
	})
	if m.pendingToolScheduleCount != 0 {
		t.Fatalf("expected scheduling cleared")
	}
	if m.runningTool != "Bash" {
		t.Fatalf("runningTool=%q", m.runningTool)
	}
	if !strings.Contains(m.runningToolLine, "Bash") {
		t.Fatalf("runningToolLine=%q", m.runningToolLine)
	}
	line := m.renderBusyAnimationLine()
	raw := stripANSI(line)
	if strings.Contains(raw, "scheduling") {
		t.Fatalf("should not still show scheduling: %q", line)
	}
	if !strings.Contains(raw, "echo") && !strings.Contains(raw, "Bash") {
		t.Fatalf("busy line should show tool summary: %q", line)
	}
}

func TestApplyKernel_ToolResultClearsRunningLine(t *testing.T) {
	t.Parallel()
	m := &model{
		width:           80,
		busy:            true,
		runningTool:     "Glob",
		runningToolLine: "Glob: *.go",
	}
	m.applyKernel(core.Event{
		Kind:           core.KindToolResult,
		ToolName:       "Glob",
		ToolResultText: "ok",
	})
	if m.runningTool != "" || m.runningToolLine != "" {
		t.Fatalf("expected cleared, tool=%q line=%q", m.runningTool, m.runningToolLine)
	}
}

func TestApplyKernel_SubTaskDepthIndentAndDepthTracking(t *testing.T) {
	t.Parallel()
	m := &model{width: 100}
	m.applyKernel(core.Event{
		Kind:         core.KindToolCall,
		SubTaskDepth: 2,
		ToolName:     "Glob",
		ToolArgs:     map[string]any{},
		ToolArgsJSON: "{}",
	})
	if m.kernelSubTaskDepth != 2 {
		t.Fatalf("kernelSubTaskDepth=%d want 2", m.kernelSubTaskDepth)
	}
	raw := stripANSI(m.committed.String())
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "    ") {
		t.Fatalf("want 4-space indent on first line, got %q", raw)
	}
	m.applyKernel(core.Event{Kind: core.KindTurnComplete})
	if m.kernelSubTaskDepth != 0 {
		t.Fatalf("after TurnComplete kernelSubTaskDepth=%d want 0", m.kernelSubTaskDepth)
	}
}

func TestStallTargetIntensity_BusyFalsePendingOnly(t *testing.T) {
	t.Parallel()
	m := &model{
		pendingToolScheduleCount: 1,
		busy:                     false,
		busyStart:                time.Now().Add(-5 * time.Minute),
		lastStreamChange:         time.Now().Add(-5 * time.Minute),
	}
	if m.stallTargetIntensity(time.Now()) != 0 {
		t.Fatalf("scheduling-only should suppress stall")
	}
}
