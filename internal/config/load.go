package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Load merges configuration sources into viper in this call order:
//  1. v3 .openclaude-profile.json (cwd, then $HOME; see profile_v3.go) — merged first (weakest)
//  2. openclaude.{yaml,yml,json} or --config file — merged next; overrides v3 for the same keys
//
// After that, spf13/viper resolution applies on each Get* (highest wins):
//  1. explicit viper.Set (rare)
//  2. cobra flags bound with BindPFlag (e.g. --provider, --model, --base-url)
//  3. environment variables (see bindViperEnv)
//  4. keys from merged config (v3 + file)
//  5. defaults implied by getters in config.go
//
// So in practice: flags beat env beat config file beat v3 profile (for the same logical key).
func Load(explicitPath string) {
	bindViperEnv()

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	MergeV3Profile(cwd, home)

	if strings.TrimSpace(explicitPath) != "" {
		viper.SetConfigFile(explicitPath)
		if err := viper.ReadInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: config: %v\n", err)
		}
		return
	}

	for _, base := range configSearchDirs(home) {
		for _, ext := range []string{"yaml", "yml", "json"} {
			p := filepath.Join(base, "openclaude."+ext)
			if _, err := os.Stat(p); err != nil {
				continue
			}
			viper.SetConfigFile(p)
			if err := viper.ReadInConfig(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "openclaude: config %s: %v\n", p, err)
				return
			}
			return
		}
	}
}

func configSearchDirs(home string) []string {
	out := []string{"."}
	if home != "" {
		out = append(out, filepath.Join(home, ".config", "openclaude"))
	}
	return out
}

func bindViperEnv() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
	_ = viper.BindEnv("provider.base_url", "OPENAI_BASE_URL")
	_ = viper.BindEnv("provider.model", "OPENAI_MODEL")
	_ = viper.BindEnv("provider.name", "OPENCLAUDE_PROVIDER")
	_ = viper.BindEnv("ollama.host", "OLLAMA_HOST")
	_ = viper.BindEnv("ollama.model", "OLLAMA_MODEL")
	_ = viper.BindEnv("gemini.api_key", "GEMINI_API_KEY", "GOOGLE_API_KEY")
	_ = viper.BindEnv("gemini.model", "GEMINI_MODEL")
	_ = viper.BindEnv("gemini.base_url", "GEMINI_BASE_URL")
	_ = viper.BindEnv("session.name", "OPENCLAUDE_SESSION")
	_ = viper.BindEnv("session.resume_last", "OPENCLAUDE_RESUME")
	_ = viper.BindEnv("session.dir", "OPENCLAUDE_SESSION_DIR")
	_ = viper.BindEnv("session.disabled", "OPENCLAUDE_NO_SESSION")
	_ = viper.BindEnv("session.compact_token_threshold", "OPENCLAUDE_SESSION_COMPACT_TOKEN_THRESHOLD")
	_ = viper.BindEnv("session.summarize_over_threshold", "OPENCLAUDE_SESSION_SUMMARIZE_OVER_THRESHOLD")
	_ = viper.BindEnv("session.compact_keep_messages", "OPENCLAUDE_SESSION_COMPACT_KEEP_MESSAGES")
	_ = viper.BindEnv("github.token", "GITHUB_TOKEN", "GITHUB_PAT")
	_ = viper.BindEnv("github.model", "GITHUB_MODEL")
	_ = viper.BindEnv("github.base_url", "GITHUB_BASE_URL")
	_ = viper.BindEnv("agent_routing.task_model", "OPENCLAUDE_AGENT_TASK_MODEL")
}
