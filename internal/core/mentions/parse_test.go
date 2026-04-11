package mentions

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAtMentionedFileLines(t *testing.T) {
	p, ls, le := ParseAtMentionedFileLines("foo.txt#L3-5")
	if p != "foo.txt" || ls != 3 || le != 5 {
		t.Fatalf("got %q %d %d", p, ls, le)
	}
	p, ls, le = ParseAtMentionedFileLines("foo.txt#L7")
	if p != "foo.txt" || ls != 7 || le != 7 {
		t.Fatalf("single line: got %q %d %d", p, ls, le)
	}
	p, ls, le = ParseAtMentionedFileLines("bar.go#heading")
	if p != "bar.go" || ls != 0 || le != 0 {
		t.Fatalf("non-L fragment: got %q %d %d", p, ls, le)
	}
}

func TestExtractMCPResourceMentions(t *testing.T) {
	s := "see @srv:res/foo and @C:\\Users\\x\\y.txt end"
	got := ExtractMCPResourceMentions(s)
	want := []string{"srv:res/foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExtractFileSpecs_skipsMCPAndMcpPrefix(t *testing.T) {
	s := `read @srv:uri/here and @mcp:x @./local.go @"a b.txt"`
	got := ExtractFileSpecs(s)
	if len(got) != 2 {
		t.Fatalf("want 2 file specs, got %#v", got)
	}
	foundLocal := false
	foundQuoted := false
	for _, x := range got {
		if strings.HasSuffix(x.Path, "local.go") {
			foundLocal = true
		}
		if x.Path == "a b.txt" {
			foundQuoted = true
		}
	}
	if !foundLocal || !foundQuoted {
		t.Fatalf("got %#v", got)
	}
}

func TestExtractFileSpecs_skipsAgent(t *testing.T) {
	s := `@agent-reviewer fix @foo.go`
	got := ExtractFileSpecs(s)
	if len(got) != 1 || got[0].Path != "foo.go" {
		t.Fatalf("got %#v", got)
	}
}
