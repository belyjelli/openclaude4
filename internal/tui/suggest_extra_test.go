package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSlashLeadingTrim(t *testing.T) {
	l, tr := slashLeadingTrim("  /foo")
	if l != 2 || tr != "/foo" {
		t.Fatalf("got %d %q", l, tr)
	}
	l, tr = slashLeadingTrim("/bar")
	if l != 0 || tr != "/bar" {
		t.Fatalf("got %d %q", l, tr)
	}
}

func TestArgWordBoundsInTrimmed(t *testing.T) {
	tr := "/session list rest"
	sp := strings.IndexByte(tr, ' ')
	ws, we := argWordBoundsInTrimmed(tr, sp)
	if tr[ws:we] != "list" {
		t.Fatalf("got %q", tr[ws:we])
	}
	tr2 := "/session   li"
	sp2 := strings.IndexByte(tr2, ' ')
	ws2, we2 := argWordBoundsInTrimmed(tr2, sp2)
	if tr2[ws2:we2] != "li" {
		t.Fatalf("got %q", tr2[ws2:we2])
	}
}

func TestTokenAtCursor(t *testing.T) {
	v := `echo ./foo/bar`
	pos := strings.Index(v, "foo")
	s, e, tok := tokenAtCursor(v, pos)
	if tok != "./foo/bar" {
		t.Fatalf("token %q", tok)
	}
	if s >= e {
		t.Fatal("bad span")
	}
}

func TestSkillNameMatches(t *testing.T) {
	got := skillNameMatches("a", []string{"Alpha", "beta", "Alpha"})
	if !slices.Equal(got, []string{"Alpha"}) {
		t.Fatalf("got %v", got)
	}
}

func TestPathCompletionMatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "apple.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "apps"), 0o700); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	matches := pathCompletionMatches("ap")
	if len(matches) < 2 {
		t.Fatalf("got %v", matches)
	}
}
