package toolpolicy

import "testing"

func TestEngine_BashPrefix(t *testing.T) {
	e := NewEngine([]string{"Bash(git:*)"}, nil)
	o, ok, tag := e.Eval("Bash", map[string]any{"command": "git status"})
	if !ok || !o.Allow || tag != "policy_allow" {
		t.Fatalf("git status: ok=%v allow=%v tag=%q", ok, o.Allow, tag)
	}
	o2, ok2, _ := e.Eval("Bash", map[string]any{"command": "npm install"})
	if ok2 && o2.Allow {
		t.Fatal("npm should not match Bash(git:*)")
	}
}

func TestEngine_DenyWins(t *testing.T) {
	e := NewEngine([]string{"Bash"}, []string{"Bash(rm:*)"})
	o, ok, tag := e.Eval("Bash", map[string]any{"command": "rm -rf ./x"})
	if !ok || o.Allow || tag != "policy_deny" {
		t.Fatalf("deny: ok=%v allow=%v tag=%q", ok, o.Allow, tag)
	}
}

func TestEngine_FilePrefix(t *testing.T) {
	e := NewEngine([]string{"FileEdit(src/*)"}, nil)
	o, ok, _ := e.Eval("FileEdit", map[string]any{"file_path": "src/foo.go", "old_string": "a", "new_string": "b"})
	if !ok || !o.Allow {
		t.Fatalf("expected allow, got %+v ok=%v", o, ok)
	}
}

func TestSuggestedAllowRule(t *testing.T) {
	s := SuggestedAllowRule("Bash", map[string]any{"command": "go test ./..."})
	if s != "Bash(go:*)" {
		t.Fatalf("got %q", s)
	}
}
