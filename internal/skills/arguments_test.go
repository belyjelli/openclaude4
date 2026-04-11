package skills

import (
	"strings"
	"testing"
)

func TestParseArgumentsHashComment(t *testing.T) {
	t.Parallel()
	got := ParseArguments("foo bar # baz")
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("got %#v", got)
	}
}

func TestSubstituteArguments(t *testing.T) {
	t.Parallel()
	body := "X=$ARGUMENTS\nA=$0 B=$1\n$nope"
	out := SubstituteArguments(body, `hello "w world"`, true, nil)
	if !strings.Contains(out, "X=hello \"w world\"") {
		t.Fatalf("ARGUMENTS: %q", out)
	}
	if !strings.Contains(out, "A=hello") || !strings.Contains(out, "B=w world") {
		t.Fatalf("positional: %q", out)
	}
	if !strings.Contains(out, "$nope") {
		t.Fatalf("should keep unknown $nope: %q", out)
	}
}

func TestSubstituteNamedArgs(t *testing.T) {
	t.Parallel()
	body := "Hello $name and $other"
	out := SubstituteArguments(body, "Alice Bob", true, []string{"name", "other"})
	if out != "Hello Alice and Bob" {
		t.Fatalf("got %q", out)
	}
}
