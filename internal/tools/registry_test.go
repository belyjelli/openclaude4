package tools

import "testing"

func TestCloneRegistryOmit(t *testing.T) {
	r := NewRegistry()
	r.Register(FileRead{})
	r.Register(Bash{})

	child := CloneRegistryOmit(r, "Bash")
	if _, ok := child.Get("Bash"); ok {
		t.Fatal("child should omit Bash")
	}
	if _, ok := child.Get("FileRead"); !ok {
		t.Fatal("child should keep FileRead")
	}
}

func TestCloneRegistryAllow(t *testing.T) {
	r := NewRegistry()
	r.Register(FileRead{})
	r.Register(Bash{})
	r.Register(Grep{})

	all := CloneRegistryAllow(r, nil)
	if len(all.List()) != 3 {
		t.Fatalf("empty allow: want 3 tools, got %d", len(all.List()))
	}

	sub := CloneRegistryAllow(r, []string{"FileRead", "Unknown"})
	if _, ok := sub.Get("FileRead"); !ok {
		t.Fatal("expected FileRead")
	}
	if _, ok := sub.Get("Bash"); ok {
		t.Fatal("Bash should be omitted")
	}
}
