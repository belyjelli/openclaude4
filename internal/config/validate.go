package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Known providers accepted by [ProviderName] / the CLI (codex is recognized but not wired yet).
var knownProviders = map[string]struct{}{
	"":        {},
	"openai":  {},
	"ollama":  {},
	"gemini":  {},
	"github":  {},
	"codex":   {},
}

// Validate checks merged config for unsupported values. Call after [Load] (and after cobra has bound flags).
func Validate() error {
	raw := strings.ToLower(strings.TrimSpace(viper.GetString("provider.name")))
	if _, ok := knownProviders[raw]; !ok {
		return fmt.Errorf("unknown provider %q (use openai, ollama, gemini, github, or codex)", viper.GetString("provider.name"))
	}
	return nil
}
