package tools

import "testing"

func TestIsGHSafeReadOnlyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{`gh pr view 1 --json title`, true},
		{`gh pr list --limit 5`, true},
		{`gh auth status`, true},
		{`gh search repos golang --limit 3`, true},
		{`/usr/bin/gh pr view 2`, true},
		{`gh pr create --title x`, false},
		{`gh pr view 1; rm -rf /`, false},
		{`gh pr view 1 --repo evil.com/secret/x`, false},
		{`echo gh pr view 1`, false},
		{`gh pr view $(id)`, false},
	}
	for _, tt := range tests {
		if got := IsGHSafeReadOnlyCommand(tt.cmd); got != tt.want {
			t.Errorf("IsGHSafeReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestShellSplit(t *testing.T) {
	got, err := shellSplit(`gh pr view "my title" --json title`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gh", "pr", "view", "my title", "--json", "title"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %q want %q", i, got[i], want[i])
		}
	}
}
