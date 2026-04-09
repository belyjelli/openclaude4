package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestPaperCLI_argvValidation(t *testing.T) {
	_, err := (PaperCLI{}).Execute(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "argv") {
		t.Fatalf("want argv required error, got %v", err)
	}

	_, err = (PaperCLI{}).Execute(context.Background(), map[string]any{
		"argv": []any{"ok", 1},
	})
	if err == nil || !strings.Contains(err.Error(), "string") {
		t.Fatalf("want argv type error, got %v", err)
	}
}

func TestPaperCLI_versionIfAvailable(t *testing.T) {
	if !PaperCLIRegistered() {
		t.Skip("papercli not on PATH and no OPENCLAUDE_PAPERCLI / PAPERCLI_BIN")
	}
	ctx := WithWorkDir(context.Background(), t.TempDir())
	out, err := (PaperCLI{}).Execute(ctx, map[string]any{
		"argv": []any{"version"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "papercli") && !strings.Contains(strings.ToLower(out), "version") {
		t.Fatalf("unexpected papercli version output: %q", out)
	}
}

func TestPaperCLI_resolveUsesEnv(t *testing.T) {
	echoPath, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("no echo in PATH")
	}
	t.Setenv("OPENCLAUDE_PAPERCLI", echoPath)
	t.Setenv("PATH", "") // still resolve binary via env, not PATH

	if !PaperCLIRegistered() {
		t.Fatal("expected registration with OPENCLAUDE_PAPERCLI")
	}
	ctx := WithWorkDir(context.Background(), t.TempDir())
	out, err := (PaperCLI{}).Execute(ctx, map[string]any{
		"argv": []string{"papercli-env-ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "papercli-env-ok") {
		t.Fatalf("want echo output, got %q", out)
	}
}
