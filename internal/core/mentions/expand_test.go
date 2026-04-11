package mentions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/tools"
)

func TestExpandUserText_fileMention(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(p, []byte("a\nb\nc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := tools.WithWorkDir(context.Background(), dir)
	out, err := ExpandUserText(ctx, "see @hello.txt#L2", Deps{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "b") || !strings.Contains(out, "### Attached") {
		t.Fatalf("got %q", out)
	}
}

func TestExpandUserText_budget(t *testing.T) {
	dir := t.TempDir()
	buf400 := strings.Repeat("x", 400*1024)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(buf400), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := tools.WithWorkDir(context.Background(), dir)
	_, err := ExpandUserText(ctx, "x @a.txt @b.txt", Deps{})
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestExpandUserText_agentInjection(t *testing.T) {
	ctx := tools.WithWorkDir(context.Background(), t.TempDir())
	agents := []config.AgentProfile{{Type: "rev", Instructions: "Be thorough."}}
	out, err := ExpandUserText(ctx, "please @agent-rev", Deps{Agents: agents})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "### Agent: rev") || !strings.Contains(out, "please @agent-rev") {
		t.Fatalf("got %q", out)
	}
}

func TestExpandUserText_agentUnknown(t *testing.T) {
	ctx := tools.WithWorkDir(context.Background(), t.TempDir())
	agents := []config.AgentProfile{{Type: "x", Instructions: "y"}}
	_, err := ExpandUserText(ctx, "@agent-z", Deps{Agents: agents})
	if err == nil {
		t.Fatal("expected error")
	}
}
