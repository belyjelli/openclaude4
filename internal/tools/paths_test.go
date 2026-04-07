package tools

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveUnderWorkdir_AllowsInside(t *testing.T) {
	root := t.TempDir()
	ctx := WithWorkDir(context.Background(), root)
	sub := filepath.Join(root, "a", "b.txt")
	got, err := resolveUnderWorkdir(ctx, filepath.Join("a", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != sub {
		t.Fatalf("got %q want %q", got, sub)
	}
}

func TestResolveUnderWorkdir_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	ctx := WithWorkDir(context.Background(), root)
	_, err := resolveUnderWorkdir(ctx, "..")
	if err == nil {
		t.Fatal("expected error for ..")
	}
}
