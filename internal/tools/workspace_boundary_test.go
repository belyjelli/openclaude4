package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUnderWorkdir_allowsInsideRelative(t *testing.T) {
	root := t.TempDir()
	ctx := WithWorkDir(context.Background(), root)
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveUnderWorkdir(ctx, filepath.Join("a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	if got != sub {
		t.Fatalf("got %q, want %q", got, sub)
	}
}

func TestResolveUnderWorkdir_allowsAbsoluteInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	ctx := WithWorkDir(context.Background(), root)
	target := filepath.Join(root, "x.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveUnderWorkdir(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("got %q, want %q", got, target)
	}
}

func TestResolveUnderWorkdir_rejectsParentTraversal(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, "SECRET")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)

	relUp := filepath.Join("..", "SECRET")
	if _, err := resolveUnderWorkdir(ctx, relUp); err == nil {
		t.Fatalf("expected error for %q", relUp)
	} else if !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("error should mention escape: %v", err)
	}
}

func TestResolveUnderWorkdir_rejectsAbsoluteOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	if _, err := resolveUnderWorkdir(ctx, outside); err == nil {
		t.Fatal("expected error for absolute path outside workspace root")
	} else if !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("error should mention escape: %v", err)
	}
}

func TestResolveUnderWorkdir_rejectsCleanedTraversalInRelative(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	// Join(inner, "sub/../..") ends up at root, one level above inner.
	p := filepath.Join("sub", "..", "..", "SECRET")
	if _, err := resolveUnderWorkdir(ctx, p); err == nil {
		t.Fatalf("expected error for %q", p)
	}
}

func TestFileRead_rejectsTraversal(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, "SECRET")
	if err := os.WriteFile(secret, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	tool := FileRead{}
	_, err := tool.Execute(ctx, map[string]any{"file_path": filepath.Join("..", "SECRET")})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("FileRead: want escapes error, got %v", err)
	}
}

func TestFileWrite_rejectsTraversal(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	tool := FileWrite{}
	_, err := tool.Execute(ctx, map[string]any{
		"file_path": filepath.Join("..", "evil.txt"),
		"content":   "pwn",
	})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("FileWrite: want escapes error, got %v", err)
	}
	// Ensure file was not created next to inner
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("evil.txt should not exist outside workspace")
	}
}

func TestFileEdit_rejectsTraversal(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, "victim.txt")
	if err := os.WriteFile(secret, []byte("ORIGINAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	tool := FileEdit{}
	_, err := tool.Execute(ctx, map[string]any{
		"file_path": filepath.Join("..", "victim.txt"),
		"old_string": "ORIGINAL",
		"new_string": "MODIFIED",
	})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("FileEdit: want escapes error, got %v", err)
	}
	b, _ := os.ReadFile(secret)
	if string(b) != "ORIGINAL" {
		t.Fatalf("file was modified: %q", b)
	}
}

func TestGrep_rejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := WithWorkDir(context.Background(), inner)
	tool := Grep{}
	_, err := tool.Execute(ctx, map[string]any{
		"pattern": "x",
		"path":    filepath.Join("..", ".."),
	})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("Grep: want escapes error, got %v", err)
	}
}
