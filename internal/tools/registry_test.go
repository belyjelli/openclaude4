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
