package bashv2

// Config controls sandbox backends, auditing, and output shaping for bash v2.
type Config struct {
	// DarwinSandbox is "best_effort" (direct exec if sandbox-exec missing), "required"
	// (Gate/Execute error when sandbox-exec not available), or "off" (admin-only; logs loudly).
	DarwinSandbox string `mapstructure:"darwinSandbox" yaml:"darwinSandbox"`
	// LinuxUseBubblewrap when true tries bwrap(1) on PATH; if unavailable and StrictLinuxSandbox
	// is true, Gate returns deny.
	LinuxUseBubblewrap bool `mapstructure:"linuxUseBubblewrap" yaml:"linuxUseBubblewrap"`
	// StrictLinuxSandbox denies execution when bwrap was requested but is not available.
	StrictLinuxSandbox bool `mapstructure:"strictLinuxSandbox" yaml:"strictLinuxSandbox"`
	// SandboxDisabled skips OS-level sandboxing (admin break-glass).
	SandboxDisabled bool `mapstructure:"sandboxDisabled" yaml:"sandboxDisabled"`
	// AuditLogPath when non-empty appends JSONL audit lines for each Gate/Execute.
	AuditLogPath string `mapstructure:"auditLogPath" yaml:"auditLogPath"`
	// InlineOutputMaxBytes is the max size returned inline to the model (default 30KiB).
	InlineOutputMaxBytes int `mapstructure:"inlineOutputMaxBytes" yaml:"inlineOutputMaxBytes"`
	// MaxTimeoutSeconds caps tool timeout (default 600).
	MaxTimeoutSeconds float64 `mapstructure:"maxTimeoutSeconds" yaml:"maxTimeoutSeconds"`
}

// DefaultConfig returns conservative defaults.
func DefaultConfig() Config {
	return Config{
		DarwinSandbox:        "best_effort",
		LinuxUseBubblewrap:   true,
		StrictLinuxSandbox:   false,
		SandboxDisabled:      false,
		AuditLogPath:         "",
		InlineOutputMaxBytes: 30 * 1024,
		MaxTimeoutSeconds:    600,
	}
}
