package tools

import "testing"

func TestIsConnectSafeReadOnlyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{`connect list`, true},
		{`connect list -d`, true},
		{`connect ls --detailed`, true},
		{`connect -v list`, true},
		{`connect -c /tmp/c.toml list`, true},
		{`connect version`, true},
		{`connect show-config`, true},
		{`connect showconfig`, true},
		{`/Users/me/toolings/connect-cli/target/release/connect list`, true},
		{`connect list evil`, false},
		{`connect import servers.txt`, false},
		{`connect connect web1`, false},
		{`connect web1`, false},
		{`connect list; rm -rf /`, false},
		{`echo connect list`, false},
		{`connect list $(id)`, false},
	}
	for _, tt := range tests {
		if got := IsConnectSafeReadOnlyCommand(tt.cmd); got != tt.want {
			t.Errorf("IsConnectSafeReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsBashReadOnlyNoConfirm_IncludesConnect(t *testing.T) {
	if !IsBashReadOnlyNoConfirm(`connect list`) {
		t.Fatal("expected connect list to be read-only no-confirm")
	}
	if IsBashReadOnlyNoConfirm(`connect import x`) {
		t.Fatal("import must not be read-only no-confirm")
	}
}
