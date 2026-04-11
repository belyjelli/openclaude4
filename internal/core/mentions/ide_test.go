package mentions

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendSyntheticAtMentions(t *testing.T) {
	wd := "/proj"
	got := AppendSyntheticAtMentions("hi", wd, []SyntheticAtPath{
		{Path: "/proj/sub/a.go", LineStart: 3, LineEnd: 5},
	})
	if !strings.Contains(got, "hi") || !strings.Contains(got, "@sub/a.go#L3-5") {
		t.Fatalf("got %q", got)
	}
}

func TestAppendSyntheticAtMentions_emptyUser(t *testing.T) {
	wd, _ := filepath.Abs(t.TempDir())
	p := filepath.Join(wd, "only.go")
	got := AppendSyntheticAtMentions("", wd, []SyntheticAtPath{{Path: p}})
	if got != "@only.go" {
		t.Fatalf("got %q", got)
	}
}
