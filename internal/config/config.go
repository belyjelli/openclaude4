package config

import (
	"strings"

	"github.com/spf13/viper"
)

const defaultModel = "gpt-4o-mini"

const defaultOllamaModel = "llama3.2"

const defaultGeminiModel = "gemini-2.0-flash"

// DefaultGeminiOpenAIBase is Google's OpenAI-compatible Gemini endpoint (v3 parity).
const DefaultGeminiOpenAIBase = "https://generativelanguage.googleapis.com/v1beta/openai"

// ProviderName returns the active backend.
func ProviderName() string {
	v := strings.ToLower(strings.TrimSpace(viper.GetString("provider.name")))
	switch v {
	case "ollama", "gemini", "openai", "codex":
		return v
	case "":
		return "openai"
	default:
		return v
	}
}

// APIKey returns the OpenAI-compatible API key (file key openai.api_key / env OPENAI_API_KEY).
func APIKey() string {
	if v := viper.GetString("openai.api_key"); v != "" {
		return v
	}
	return ""
}

// BaseURL returns optional OpenAI-compatible API base URL (empty = SDK default).
func BaseURL() string {
	if v := viper.GetString("OPENAI_BASE_URL"); v != "" {
		return v
	}
	if v := viper.GetString("provider.base_url"); v != "" {
		return v
	}
	return ""
}

// Model returns the chat model for the active provider.
func Model() string {
	switch ProviderName() {
	case "ollama":
		return OllamaModel()
	case "gemini":
		return GeminiModel()
	default:
		return openAIModel()
	}
}

func openAIModel() string {
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	return defaultModel
}

// OllamaModel returns the Ollama tag (ollama.model / OLLAMA_MODEL / provider.model / default).
func OllamaModel() string {
	if v := viper.GetString("ollama.model"); v != "" {
		return v
	}
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	return defaultOllamaModel
}

// OllamaChatBase returns the OpenAI-compatible chat base URL for Ollama (…/v1).
func OllamaChatBase() string {
	raw := strings.TrimSpace(viper.GetString("ollama.host"))
	if raw == "" {
		raw = "http://127.0.0.1:11434"
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/v1") {
		return raw
	}
	return raw + "/v1"
}

// GeminiAPIKey returns GEMINI_API_KEY or GOOGLE_API_KEY (merged via viper).
func GeminiAPIKey() string {
	if v := viper.GetString("gemini.api_key"); v != "" {
		return v
	}
	return ""
}

// GeminiBaseURL returns the OpenAI-compatible base URL for Gemini (no trailing slash).
func GeminiBaseURL() string {
	if v := strings.TrimSpace(viper.GetString("gemini.base_url")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(DefaultGeminiOpenAIBase, "/")
}

// GeminiModel returns the Gemini model id.
func GeminiModel() string {
	if v := viper.GetString("gemini.model"); v != "" {
		return v
	}
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	return defaultGeminiModel
}
