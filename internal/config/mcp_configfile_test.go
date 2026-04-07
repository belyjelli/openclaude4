package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendMCPServerToConfigFile_NewYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "openclaude.yaml")
	srv := MCPServer{Name: "fs", Command: []string{"npx", "-y", "pkg"}, Approval: "ask"}
	if err := AppendMCPServerToConfigFile(p, srv); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "fs") || !strings.Contains(s, "npx") || !strings.Contains(s, "mcp") {
		t.Fatalf("unexpected file:\n%s", s)
	}
}

func TestAppendMCPServerToConfigFile_Duplicate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "openclaude.yaml")
	srv := MCPServer{Name: "fs", Command: []string{"npx"}, Approval: "never"}
	if err := AppendMCPServerToConfigFile(p, srv); err != nil {
		t.Fatal(err)
	}
	err := AppendMCPServerToConfigFile(p, srv)
	if !errors.Is(err, ErrMCPNameExists) {
		t.Fatalf("want ErrMCPNameExists, got %v", err)
	}
}

func TestAppendMCPServerToConfigFile_AppendSecond(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "openclaude.yaml")
	if err := AppendMCPServerToConfigFile(p, MCPServer{Name: "a", Command: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if err := AppendMCPServerToConfigFile(p, MCPServer{Name: "b", Command: []string{"y"}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "a") || !strings.Contains(string(b), "b") {
		t.Fatalf("%s", b)
	}
}
