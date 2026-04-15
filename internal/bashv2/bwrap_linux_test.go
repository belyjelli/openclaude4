//go:build linux

package bashv2

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxBwrapEcho(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed")
	}
	tmp := t.TempDir()
	cfg := DefaultConfig()
	cfg.LinuxUseBubblewrap = true
	cfg.SandboxDisabled = false
	s := NewSession(SessionOpts{Config: cfg})
	ctx := context.Background()
	out, err := s.Execute(ctx, ExecuteInput{
		Workspace: tmp,
		Args: map[string]any{
			"command": "echo hi",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hi") {
		t.Fatalf("output %q", out)
	}
}

func TestLinuxBwrapWorkspaceWritable(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed")
	}
	tmp := t.TempDir()
	cfg := DefaultConfig()
	cfg.LinuxUseBubblewrap = true
	s := NewSession(SessionOpts{Config: cfg})
	ctx := context.Background()
	out, err := s.Execute(ctx, ExecuteInput{
		Workspace: tmp,
		Args: map[string]any{
			"command": "touch ./ok && echo done",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "ok")); err != nil {
		t.Fatalf("touch failed: %v out=%q", err, out)
	}
}
