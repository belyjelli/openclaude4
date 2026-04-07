package config

import (
	"strings"

	"github.com/spf13/viper"
)

const defaultModel = "gpt-4o-mini"

// Load reads provider settings from the environment (and optional config file later).
func Load() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// APIKey returns the OpenAI-compatible API key (often OPENAI_API_KEY).
func APIKey() string {
	return viper.GetString("OPENAI_API_KEY")
}

// BaseURL returns an optional override for the OpenAI-compatible API base URL.
func BaseURL() string {
	if v := viper.GetString("OPENAI_BASE_URL"); v != "" {
		return v
	}
	if v := viper.GetString("provider.base_url"); v != "" {
		return v
	}
	return ""
}

// Model returns the chat model name.
func Model() string {
	if v := viper.GetString("OPENAI_MODEL"); v != "" {
		return v
	}
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	return defaultModel
}
