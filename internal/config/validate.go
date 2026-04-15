package config

import (
	"fmt"
	"strings"

	"github.com/gitlawb/openclaude4/internal/providererrs"
	"github.com/spf13/viper"
)

// Known providers accepted by [ProviderName] / the CLI (codex fails Validate with providererrs.ErrCodexNotImplemented).
var knownProviders = map[string]struct{}{
	"":           {},
	"openai":     {},
	"ollama":     {},
	"gemini":     {},
	"github":     {},
	"openrouter": {},
	"codex":      {},
}

// Validate checks merged config for unsupported values. Call after [Load] (and after cobra has bound flags).
func Validate() error {
	raw := strings.ToLower(strings.TrimSpace(viper.GetString("provider.name")))
	if _, ok := knownProviders[raw]; !ok {
		return fmt.Errorf("unknown provider %q (use openai, ollama, gemini, github, openrouter, or codex)", viper.GetString("provider.name"))
	}
	if raw == "codex" {
		return providererrs.ErrCodexNotImplemented
	}
	ds := strings.ToLower(strings.TrimSpace(viper.GetString("bashv2.darwinSandbox")))
	switch ds {
	case "", "best_effort", "required", "off":
	default:
		return fmt.Errorf("bashv2.darwinSandbox: unknown value %q (use best_effort, required, or off)", viper.GetString("bashv2.darwinSandbox"))
	}
	return nil
}
