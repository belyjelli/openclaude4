package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSkillMarkdown(t *testing.T) {
	t.Parallel()
	raw := `---
name: demo
description: A test skill
---
# Do this

Hello.
`
	n, d, b, err := parseSkillMarkdown([]byte(raw), "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if n != "demo" {
		t.Fatalf("name %q", n)
	}
	if d != "A test skill" {
		t.Fatalf("desc %q", d)
	}
	if b == "" || !strings.Contains(b, "Hello") {
		t.Fatalf("body %q", b)
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "mine")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(md, []byte("---\nname: alpha\ndescription: d\n---\nbody"), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := Load([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 1 {
		t.Fatalf("len %d", cat.Len())
	}
	e, ok := cat.Get("alpha")
	if !ok || e.Body != "body" {
		t.Fatalf("%+v", e)
	}
	e2, ok2 := cat.GetFold("ALPHA")
	if !ok2 || e2.Name != "alpha" {
		t.Fatalf("GetFold: %+v ok=%v", e2, ok2)
	}
}
