package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// DescribeEffectiveConfig prints merged config sources and non-secret effective values (for /config).
func DescribeEffectiveConfig(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	_, _ = fmt.Fprintln(w, "Precedence: CLI flags > environment > openclaude.yaml/json > v3 .openclaude-profile.json > defaults")
	_, _ = fmt.Fprintln(w, "(see internal/config/load.go)")

	if p := strings.TrimSpace(viper.ConfigFileUsed()); p != "" {
		_, _ = fmt.Fprintf(w, "Merged config file: %s\n", p)
	} else {
		_, _ = fmt.Fprintln(w, "Merged config file: (none — only v3 profile, env, flags, defaults)")
	}

	if mp := strings.TrimSpace(MergedV3ProfilePath); mp != "" {
		_, _ = fmt.Fprintf(w, "v3 profile merged: %s\n", mp)
	} else {
		_, _ = fmt.Fprintln(w, "v3 profile merged: (none)")
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	_, _ = fmt.Fprintln(w, "Standard file search (first existing file wins at load):")
	for _, base := range configSearchDirs(home) {
		for _, ext := range []string{"yaml", "yml", "json"} {
			name := filepath.Join(base, "openclaude."+ext)
			rel := name
			if cwd != "" {
				if r, err := filepath.Rel(cwd, name); err == nil && !strings.HasPrefix(r, "..") {
					rel = r
				}
			}
			st, err := os.Stat(name)
			if err != nil {
				_, _ = fmt.Fprintf(w, "  %s  (missing)\n", name)
				continue
			}
			if st.IsDir() {
				_, _ = fmt.Fprintf(w, "  %s  (directory — skipped)\n", name)
				continue
			}
			_, _ = fmt.Fprintf(w, "  %s  [exists]  (%s)\n", name, rel)
		}
	}

	if wp, err := WritableConfigPath(); err != nil {
		_, _ = fmt.Fprintf(w, "Writable config path (mcp add, etc.): (error) %v\n", err)
	} else {
		_, _ = fmt.Fprintf(w, "Writable config path (mcp add, etc.): %s\n", wp)
	}

	_, _ = fmt.Fprintln(w, "Effective (non-secret):")
	_, _ = fmt.Fprintf(w, "  provider: %s\n", ProviderName())
	_, _ = fmt.Fprintf(w, "  model: %s\n", Model())
	_, _ = fmt.Fprintf(w, "  sessions: disabled=%v  dir=%s\n", SessionDisabled(), EffectiveSessionDir())

	srv := MCPServers()
	if len(srv) == 0 {
		_, _ = fmt.Fprintln(w, "  mcp.servers: (none)")
		return
	}
	_, _ = fmt.Fprintf(w, "  mcp.servers (%d):\n", len(srv))
	for _, s := range srv {
		ap := NormalizeMCPApproval(s.Approval)
		_, _ = fmt.Fprintf(w, "    - %s  approval=%s\n", s.Name, ap)
	}
}
