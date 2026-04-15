package validate

import (
	"strings"
	"testing"
)

func TestBlockedSubstrings(t *testing.T) {
	t.Parallel()
	v := blockedSubstrings{}
	tests := []struct {
		unit   string
		want   Verdict
		substr string // substring of reason when want==Fail
	}{
		{"ls", Pass, ""},
		{"rm -rf ./build", Pass, ""},
		{"rm -rf /", Fail, "rm -rf /"},
		{"echo && rm -rf /*", Fail, "rm -rf"},
		{"sudo mkfs.ext4 /dev/sda", Fail, "mkfs."},
		{"dd if=/dev/zero of=x", Fail, "dd if="},
		{"echo :(){ :|:& };:", Fail, ":(){"},
		{"chmod -R 777 /etc", Fail, "chmod -R 777 /"},
		{"cat > /dev/mem", Fail, "dev/mem"},
		{"cat > /dev/kmem", Fail, "dev/kmem"},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, reason := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("Check(%q) verdict=%v reason=%q want %v", tt.unit, got, reason, tt.want)
			}
			if tt.want == Fail && tt.substr != "" && !strings.Contains(reason, tt.substr) {
				t.Fatalf("reason %q should mention %q", reason, tt.substr)
			}
		})
	}
}

func TestNoSudo(t *testing.T) {
	t.Parallel()
	v := noSudo{}
	tests := []struct {
		unit string
		want Verdict
	}{
		{"ls -la", Pass},
		{"sudo ls", Fail},
		{"prefix sudo ls", Fail},
		{"(sudo ls)", Fail},
		{"git diff HEAD", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, _ := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("got %v for %q", got, tt.unit)
			}
		})
	}
}

func TestSuspiciousEnv(t *testing.T) {
	t.Parallel()
	v := suspiciousEnv{}
	tests := []struct {
		unit string
		want Verdict
	}{
		{"export PATH=/x", Pass},
		{"LD_PRELOAD=/evil.so ls", Fail},
		{"env LD_PRELOAD=x true", Fail},
		{"BASH_ENV=~/.bashrc bash -c true", Fail},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, _ := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("got %v for %q", got, tt.unit)
			}
		})
	}
}

func TestCurlPipeShell(t *testing.T) {
	t.Parallel()
	v := curlPipeShell{}
	tests := []struct {
		unit string
		want Verdict
	}{
		{"curl -s https://x | sh", Fail},
		{"wget -O- u | bash", Fail},
		{"curl -s u | /usr/bin/env sh", Fail},
		{"curl -s https://example.com", Pass},
		{"wget -q u -O f", Pass},
		{"echo curl '|' sh", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, _ := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("got %v for %q", got, tt.unit)
			}
		})
	}
}

func TestNoChrootNsenter(t *testing.T) {
	t.Parallel()
	v := noChrootNsenter{}
	tests := []struct {
		unit string
		want Verdict
	}{
		{"chroot /mnt /bin/sh", Fail},
		{"nsenter -t 1 -m bash", Fail},
		{"echo about-chroot-in-text", Pass},
		{"./mychroot", Pass},
		// Parsed segment after && is validated separately by [parse.SplitUnits] + [Chain].
		{"chroot /x", Fail},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, _ := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("got %v for %q", got, tt.unit)
			}
		})
	}
}

func TestPosixSyntax(t *testing.T) {
	t.Parallel()
	v := posixSyntax{}
	tests := []struct {
		unit string
		want Verdict
	}{
		{"ls -la", Pass},
		{"git status", Pass},
		{"", Fail},
		{"   ", Fail},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got, _ := v.Check(tt.unit)
			if got != tt.want {
				t.Fatalf("got %v for %q", got, tt.unit)
			}
		})
	}
}

func TestDefaultChain_orderAndFail(t *testing.T) {
	t.Parallel()
	chain := DefaultChain()
	units := []string{"echo ok"}
	if verdict, _, _ := Chain(chain, units); verdict != Pass {
		t.Fatal("expected pass")
	}
	units2 := []string{"sudo true"}
	if verdict, id, _ := Chain(chain, units2); verdict != Fail || id != "no_sudo" {
		t.Fatalf("want no_sudo fail, got %v %s", verdict, id)
	}
}
