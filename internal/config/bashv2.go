package config

import (
	"strings"

	"github.com/gitlawb/openclaude4/internal/bashv2"
	"github.com/spf13/viper"
)

// BashV2 returns merged bash v2 execution settings from the bashv2: YAML key.
func BashV2() bashv2.Config {
	c := bashv2.DefaultConfig()
	var wrap struct {
		DarwinSandbox        string  `mapstructure:"darwinSandbox"`
		LinuxUseBubblewrap   *bool   `mapstructure:"linuxUseBubblewrap"`
		StrictLinuxSandbox   *bool   `mapstructure:"strictLinuxSandbox"`
		SandboxDisabled      *bool   `mapstructure:"sandboxDisabled"`
		AuditLogPath         string  `mapstructure:"auditLogPath"`
		InlineOutputMaxBytes *int    `mapstructure:"inlineOutputMaxBytes"`
		MaxTimeoutSeconds    float64 `mapstructure:"maxTimeoutSeconds"`
	}
	if err := viper.UnmarshalKey("bashv2", &wrap); err != nil {
		return c
	}
	if s := strings.TrimSpace(wrap.DarwinSandbox); s != "" {
		c.DarwinSandbox = s
	}
	if wrap.LinuxUseBubblewrap != nil {
		c.LinuxUseBubblewrap = *wrap.LinuxUseBubblewrap
	}
	if wrap.StrictLinuxSandbox != nil {
		c.StrictLinuxSandbox = *wrap.StrictLinuxSandbox
	}
	if wrap.SandboxDisabled != nil {
		c.SandboxDisabled = *wrap.SandboxDisabled
	}
	if p := strings.TrimSpace(wrap.AuditLogPath); p != "" {
		c.AuditLogPath = p
	}
	if wrap.InlineOutputMaxBytes != nil && *wrap.InlineOutputMaxBytes > 0 {
		c.InlineOutputMaxBytes = *wrap.InlineOutputMaxBytes
	}
	if wrap.MaxTimeoutSeconds > 0 {
		c.MaxTimeoutSeconds = wrap.MaxTimeoutSeconds
	}
	return c
}
