package tools

import "testing"

func TestIsNetstatSafeReadOnlyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{`netstat`, true},
		{`netstat -a`, true},
		{`netstat -an`, true},
		{`netstat -a -n`, true},
		{`/usr/sbin/netstat -rn`, true},
		{`netstat -f inet`, true},
		{`netstat -I en0`, true},
		{`netstat -s`, true},
		{`netstat -v`, true},
		{`netstat -p tcp`, false},
		{`netstat --help`, false},
		{`netstat -- -a`, false},
		{`netstat -an; id`, false},
		{`echo netstat -a`, false},
		{`netstat $(id)`, false},
	}
	for _, tt := range tests {
		if got := IsNetstatSafeReadOnlyCommand(tt.cmd); got != tt.want {
			t.Errorf("IsNetstatSafeReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsBashReadOnlyNoConfirm(t *testing.T) {
	if !IsBashReadOnlyNoConfirm(`gh pr view 1 --json title`) {
		t.Fatal("expected gh read-only to skip confirm")
	}
	if !IsBashReadOnlyNoConfirm(`netstat -an`) {
		t.Fatal("expected netstat read-only to skip confirm")
	}
	if IsBashReadOnlyNoConfirm(`gh pr create --title x`) {
		t.Fatal("expected mutating gh to require confirm")
	}
	if IsBashReadOnlyNoConfirm(`netstat -p 1`) {
		t.Fatal("expected disallowed netstat flag to require confirm")
	}
}
