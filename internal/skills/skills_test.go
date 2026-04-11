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
	ent, err := ParseSkillMarkdown([]byte(raw), "fallback", "")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Name != "demo" {
		t.Fatalf("name %q", ent.Name)
	}
	if ent.Description != "A test skill" {
		t.Fatalf("desc %q", ent.Description)
	}
	if ent.Body == "" || !strings.Contains(ent.Body, "Hello") {
		t.Fatalf("body %q", ent.Body)
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
	if !ok || e.Body != "body" || !e.SourceLocal {
		t.Fatalf("%+v", e)
	}
	e2, ok2 := cat.GetFold("ALPHA")
	if !ok2 || e2.Name != "alpha" {
		t.Fatalf("GetFold: %+v ok=%v", e2, ok2)
	}
}

func TestParseSkillMarkdownExtended(t *testing.T) {
	t.Parallel()
	raw := `---
name: ext
description: d
allowed_tools: [Read, Bash]
arguments: foo bar
context: fork
max_fork_iterations: 5
hooks:
  UserPromptSubmit:
    - hooks:
        - type: command
          command: "true"
---
Body here.
`
	ent, err := ParseSkillMarkdown([]byte(raw), "fb", "/skill/dir")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Name != "ext" {
		t.Fatalf("name %q", ent.Name)
	}
	if len(ent.AllowedTools) != 2 || ent.AllowedTools[0] != "Read" {
		t.Fatalf("allowed_tools %#v", ent.AllowedTools)
	}
	if len(ent.ArgumentNames) != 2 || ent.ArgumentNames[0] != "foo" {
		t.Fatalf("arguments %#v", ent.ArgumentNames)
	}
	if ent.Context != "fork" || ent.MaxForkIterations != 5 {
		t.Fatalf("fork meta: ctx=%q max=%d", ent.Context, ent.MaxForkIterations)
	}
	if ent.Hooks == nil {
		t.Fatal("expected hooks")
	}
	if ent.Body != "Body here." {
		t.Fatalf("body %q", ent.Body)
	}
}
