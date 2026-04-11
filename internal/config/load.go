package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ConfigExplicitPath is the absolute path passed to Load via --config, or empty when
// implicit multi-file merge was used. WritableConfigPath uses this to persist to the
// same file the user pointed at.
var ConfigExplicitPath string

// MergedHomeOpenClaudePath is the ~/.config/openclaude/openclaude.{yaml,yml,json}
// merged in the last implicit Load, if any.
var MergedHomeOpenClaudePath string

// MergedHomeOpenClaudeV4Path is the ~/.config/openclaude/openclaudev4.{yaml,yml,json}
// merged in the last implicit Load, if any (overrides home openclaude for same keys).
var MergedHomeOpenClaudeV4Path string

// MergedCwdOpenClaudePath is the cwd openclaude.{yaml,yml,json} merged in the last
// implicit Load, if any (strongest file layer).
var MergedCwdOpenClaudePath string

func resetOpenClaudeMergeState() {
	ConfigExplicitPath = ""
	MergedHomeOpenClaudePath = ""
	MergedHomeOpenClaudeV4Path = ""
	MergedCwdOpenClaudePath = ""
}

// firstConfigStemInDir returns the first existing path among stem.yaml, stem.yml, stem.json
// in dir, or "" if none exist or dir is empty.
func firstConfigStemInDir(dir, stem string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	for _, ext := range []string{"yaml", "yml", "json"} {
		p := filepath.Join(dir, stem+"."+ext)
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		return p
	}
	return ""
}

// Load merges configuration sources into viper in this call order:
//
//	Explicit --config path: v3 profile, then that single file.
//
//	Implicit (no --config): v3 profile, then (weakest to strongest among YAML files):
//	  ~/.config/openclaude/openclaude.{yaml,yml,json}
//	  ~/.config/openclaude/openclaudev4.{yaml,yml,json}
//	  ./openclaude.{yaml,yml,json}
//
// After that, spf13/viper resolution applies on each Get* (highest wins):
//  1. explicit viper.Set (rare)
//  2. cobra flags bound with BindPFlag (e.g. --provider, --model, --base-url)
//  3. environment variables (see bindViperEnv; OPENAI_MODEL binds to openai.model, not provider.model)
//  4. keys from merged config (v3 + file(s))
//  5. defaults implied by getters in config.go
//
// So in practice: flags beat env beat merged YAML beat v3 profile (for the same logical key).
func Load(explicitPath string) {
	bindViperEnv()

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	resetOpenClaudeMergeState()
	MergeV3Profile(cwd, home)

	if strings.TrimSpace(explicitPath) != "" {
		viper.SetConfigFile(explicitPath)
		if err := viper.ReadInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: config: %v\n", err)
			return
		}
		abs, err := filepath.Abs(explicitPath)
		if err != nil {
			ConfigExplicitPath = strings.TrimSpace(explicitPath)
		} else {
			ConfigExplicitPath = abs
		}
		return
	}

	loadImplicitOpenClaudeFiles(cwd, home)
}

func loadImplicitOpenClaudeFiles(cwd, home string) {
	homeDir := ""
	if home != "" {
		homeDir = filepath.Join(home, ".config", "openclaude")
	}

	var paths []string
	if p := firstConfigStemInDir(homeDir, "openclaude"); p != "" {
		MergedHomeOpenClaudePath = p
		paths = append(paths, p)
	}
	if p := firstConfigStemInDir(homeDir, "openclaudev4"); p != "" {
		MergedHomeOpenClaudeV4Path = p
		paths = append(paths, p)
	}
	cwdDir := cwd
	if cwdDir == "" {
		cwdDir = "."
	}
	if p := firstConfigStemInDir(cwdDir, "openclaude"); p != "" {
		MergedCwdOpenClaudePath = p
		paths = append(paths, p)
	}

	if len(paths) == 0 {
		return
	}

	first := true
	for _, p := range paths {
		viper.SetConfigFile(p)
		var err error
		if first {
			err = viper.ReadInConfig()
			first = false
		} else {
			err = viper.MergeInConfig()
		}
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: config %s: %v\n", p, err)
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
	_ = viper.BindEnv("openai.model", "OPENAI_MODEL")
	_ = viper.BindEnv("provider.base_url", "OPENAI_BASE_URL", "OPENROUTER_BASE_URL")
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
	_ = viper.BindEnv("openrouter.api_key", "OPENROUTER_KEY", "OPENROUTER_API_KEY")
	_ = viper.BindEnv("openrouter.provider", "OPENROUTER_PROVIDER")
	_ = viper.BindEnv("openrouter.model", "OPENROUTER_MODEL")
	_ = viper.BindEnv("agent_routing.task_model", "OPENCLAUDE_AGENT_TASK_MODEL")
	_ = viper.BindEnv("tui.busy_verbose_tokens", "OPENCLAUDE_TUI_BUSY_VERBOSE_TOKENS")
}
