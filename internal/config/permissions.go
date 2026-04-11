package config

import (
	"strings"

	"github.com/spf13/viper"
)

// PermissionsFromViper returns global allow/deny rule strings (v3-style, e.g. Bash(git:*), FileEdit).
func PermissionsFromViper() (allow, deny []string) {
	var wrap struct {
		Allow []string `mapstructure:"allow"`
		Deny  []string `mapstructure:"deny"`
	}
	if err := viper.UnmarshalKey("permissions", &wrap); err != nil {
		return nil, nil
	}
	for _, s := range wrap.Allow {
		s = strings.TrimSpace(s)
		if s != "" {
			allow = append(allow, s)
		}
	}
	for _, s := range wrap.Deny {
		s = strings.TrimSpace(s)
		if s != "" {
			deny = append(deny, s)
		}
	}
	return allow, deny
}
