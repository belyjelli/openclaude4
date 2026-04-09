package tools

import "testing"

func TestIsSsSafeReadOnlyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{`ss -l -n`, true},
		{`ss -ltn`, true},
		{`ss -tan`, true},
		{`/usr/bin/ss -s`, true},
		{`ss -f inet`, true},
		{`ss --family=inet`, true},
		{`ss -A tcp`, true},
		{`ss --help`, true},
		{`ss -V`, true},
		{`ss -4 -l`, true},
		{`ss -K`, false},
		{`ss --kill`, false},
		{`ss -D /tmp/x`, false},
		{`ss --diag=/tmp/x`, false},
		{`ss -F /etc/passwd`, false},
		{`ss -N`, false},
		{`ss --net=foo`, false},
		{`ss -- -l`, false},
		{`ss -ltn; id`, false},
		{`echo ss -l`, false},
	}
	for _, tt := range tests {
		if got := IsSsSafeReadOnlyCommand(tt.cmd); got != tt.want {
			t.Errorf("IsSsSafeReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsBashReadOnlyNoConfirm_IncludesSs(t *testing.T) {
	if !IsBashReadOnlyNoConfirm(`ss -ltn`) {
		t.Fatal("expected ss read-only to skip confirm")
	}
}
