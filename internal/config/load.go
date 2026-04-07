package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Load merges defaults, optional config file, and environment (env wins over file).
// explicitPath: if set, read exactly this file (warn on error).
// Otherwise pick the first existing file among:
//
//	./openclaude.{yaml,yml,json}
//	~/.config/openclaude/openclaude.{yaml,yml,json}
func Load(explicitPath string) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
	_ = viper.BindEnv("provider.base_url", "OPENAI_BASE_URL")
	_ = viper.BindEnv("provider.model", "OPENAI_MODEL")
	_ = viper.BindEnv("provider.name", "OPENCLAUDE_PROVIDER")
	_ = viper.BindEnv("ollama.host", "OLLAMA_HOST")
	_ = viper.BindEnv("ollama.model", "OLLAMA_MODEL")

	if strings.TrimSpace(explicitPath) != "" {
		viper.SetConfigFile(explicitPath)
		if err := viper.ReadInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "openclaude: config: %v\n", err)
		}
		return
	}

	home, _ := os.UserHomeDir()
	bases := []string{"."}
	if home != "" {
		bases = append(bases, filepath.Join(home, ".config", "openclaude"))
	}
	for _, base := range bases {
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
