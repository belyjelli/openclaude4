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
	_, _ = fmt.Fprintln(w, "Precedence: CLI flags > environment > merged openclaude YAML (cwd wins over home stack) > v3 .openclaude-profile.json > defaults")
	_, _ = fmt.Fprintln(w, "(see internal/config/load.go)")

	if p := strings.TrimSpace(ConfigExplicitPath); p != "" {
		_, _ = fmt.Fprintf(w, "Config file (--config): %s\n", p)
	} else {
		var merged []string
		for _, p := range []string{
			MergedHomeOpenClaudePath,
			MergedHomeOpenClaudeV4Path,
			MergedCwdOpenClaudePath,
		} {
			if strings.TrimSpace(p) != "" {
				merged = append(merged, p)
			}
		}
		if len(merged) == 0 {
			_, _ = fmt.Fprintln(w, "Merged openclaude config files: (none — only v3 profile, env, flags, defaults)")
		} else {
			_, _ = fmt.Fprintln(w, "Merged openclaude config files (weakest → strongest):")
			for _, p := range merged {
				_, _ = fmt.Fprintf(w, "  %s\n", p)
			}
		}
	}

	if mp := strings.TrimSpace(MergedV3ProfilePath); mp != "" {
		_, _ = fmt.Fprintf(w, "v3 profile merged: %s\n", mp)
	} else {
		_, _ = fmt.Fprintln(w, "v3 profile merged: (none)")
	}

	home, _ := os.UserHomeDir()
	_, _ = fmt.Fprintln(w, "Implicit merge candidates (home: openclaude then openclaudev4; then cwd openclaude):")
	if home != "" {
		hdir := filepath.Join(home, ".config", "openclaude")
		for _, stem := range []string{"openclaude", "openclaudev4"} {
			for _, ext := range []string{"yaml", "yml", "json"} {
				name := filepath.Join(hdir, stem+"."+ext)
				st, err := os.Stat(name)
				if err != nil {
					_, _ = fmt.Fprintf(w, "  %s  (missing)\n", name)
					continue
				}
				if st.IsDir() {
					_, _ = fmt.Fprintf(w, "  %s  (directory — skipped)\n", name)
					continue
				}
				_, _ = fmt.Fprintf(w, "  %s  [exists]\n", name)
			}
		}
	} else {
		_, _ = fmt.Fprintln(w, "  (no $HOME — home merge paths skipped)")
	}
	for _, ext := range []string{"yaml", "yml", "json"} {
		name := filepath.Join(".", "openclaude."+ext)
		st, err := os.Stat(name)
		if err != nil {
			_, _ = fmt.Fprintf(w, "  %s  (missing)\n", name)
			continue
		}
		if st.IsDir() {
			_, _ = fmt.Fprintf(w, "  %s  (directory — skipped)\n", name)
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s  [exists]\n", name)
	}

	if wp, err := WritableConfigPath(); err != nil {
		_, _ = fmt.Fprintf(w, "Writable config path (mcp add, etc.): (error) %v\n", err)
	} else {
		_, _ = fmt.Fprintf(w, "Writable config path (mcp add, etc.): %s\n", wp)
	}

	if p := strings.TrimSpace(viper.ConfigFileUsed()); p != "" && strings.TrimSpace(ConfigExplicitPath) == "" {
		_, _ = fmt.Fprintf(w, "viper.ConfigFileUsed (last merged file): %s\n", p)
	}

	_, _ = fmt.Fprintln(w, "Effective (non-secret):")
	_, _ = fmt.Fprintf(w, "  provider: %s\n", ProviderName())
	_, _ = fmt.Fprintf(w, "  model: %s\n", Model())
	if k := OpenRouterAPIKey(); k != "" {
		_, _ = fmt.Fprintf(w, "  openrouter: api_key=(set, len=%d)  provider_filter=%q\n", len(k), OpenRouterProviderFilter())
	}
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
