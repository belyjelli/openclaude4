package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteBracketedPastePaths_single(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(p, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, ok := rewriteBracketedPastePaths(p, dir)
	if !ok || !strings.Contains(out, "@doc.txt") {
		t.Fatalf("got %q ok=%v", out, ok)
	}
}

func TestRewriteBracketedPastePaths_multi(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	raw := filepath.Join(dir, "a.go") + "\n" + filepath.Join(dir, "b.go")
	out, ok := rewriteBracketedPastePaths(raw, dir)
	if !ok || !strings.Contains(out, "@a.go") || !strings.Contains(out, "@b.go") {
		t.Fatalf("got %q", out)
	}
}

func TestRewriteBracketedPastePaths_notPath(t *testing.T) {
	out, ok := rewriteBracketedPastePaths("hello world", "")
	if ok || out != "hello world" {
		t.Fatalf("got %q ok=%v", out, ok)
	}
}
