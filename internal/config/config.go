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

// DefaultOpenRouterOpenAIBase is OpenRouter's OpenAI-compatible API root.
const DefaultOpenRouterOpenAIBase = "https://openrouter.ai/api/v1"

// ProviderName returns the active backend.
func ProviderName() string {
	v := strings.ToLower(strings.TrimSpace(viper.GetString("provider.name")))
	switch v {
	case "ollama", "gemini", "openai", "codex", "github", "openrouter":
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
		return strings.TrimRight(strings.TrimSpace(v), "/")
	}
	if v := viper.GetString("provider.base_url"); v != "" {
		return strings.TrimRight(strings.TrimSpace(v), "/")
	}
	return ""
}

// BaseURLLooksLikeOpenRouter reports whether base points at OpenRouter's API host.
func BaseURLLooksLikeOpenRouter(base string) bool {
	b := strings.ToLower(strings.TrimSpace(base))
	return strings.Contains(b, "openrouter.ai")
}

// EffectiveOpenAICompatAPIKey returns OPENAI_API_KEY for chat, or OPENROUTER_KEY when the
// configured base URL targets OpenRouter and the OpenAI key is empty.
func EffectiveOpenAICompatAPIKey() string {
	if k := strings.TrimSpace(APIKey()); k != "" {
		return k
	}
	if BaseURLLooksLikeOpenRouter(BaseURL()) {
		return OpenRouterAPIKey()
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
	case "github":
		return GitHubModelsModel()
	case "openrouter":
		return OpenRouterModel()
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

// OpenRouterAPIKey returns OPENROUTER_KEY / OPENROUTER_API_KEY or openrouter.api_key (merged via viper).
// Used to list models via the OpenRouter API catalog (/model); independent of OPENAI_API_KEY.
func OpenRouterAPIKey() string {
	if v := strings.TrimSpace(viper.GetString("openrouter.api_key")); v != "" {
		return v
	}
	return ""
}

// OpenRouterProviderFilter returns OPENROUTER_PROVIDER or openrouter.provider: optional slug (e.g. "anthropic", "openai")
// to restrict OpenRouter model IDs to those with that prefix (provider/slug in model id).
func OpenRouterProviderFilter() string {
	return strings.TrimSpace(strings.ToLower(viper.GetString("openrouter.provider")))
}

// OpenRouterChatBase returns the OpenAI-compatible base URL for OpenRouter (no trailing slash).
// Uses OPENAI_BASE_URL / provider.base_url when set; otherwise DefaultOpenRouterOpenAIBase.
func OpenRouterChatBase() string {
	if v := BaseURL(); v != "" {
		return v
	}
	return strings.TrimRight(DefaultOpenRouterOpenAIBase, "/")
}

// OpenRouterModel returns the model id for provider openrouter (OpenRouter-style slugs).
func OpenRouterModel() string {
	if v := viper.GetString("openrouter.model"); v != "" {
		return v
	}
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	return "openai/gpt-4o-mini"
}

// GitHubToken returns GITHUB_TOKEN or GITHUB_PAT (merged via viper).
func GitHubToken() string {
	if v := viper.GetString("github.token"); v != "" {
		return v
	}
	return ""
}

// GitHubModelsBaseURL returns the OpenAI-compatible base URL for GitHub Models.
func GitHubModelsBaseURL() string {
	if v := strings.TrimSpace(viper.GetString("github.base_url")); v != "" {
		return strings.TrimRight(v, "/")
	}
	// GitHub Models uses the Azure-compatible endpoint
	// Pattern: https://{region}.models.ai.azure.com
	// Users can set GITHUB_BASE_URL to customize
	return ""
}

// GitHubModelsModel returns the GitHub Models model id.
func GitHubModelsModel() string {
	if v := viper.GetString("github.model"); v != "" {
		return v
	}
	if v := viper.GetString("provider.model"); v != "" {
		return v
	}
	// Default to a common GitHub Models model
	return "gpt-4o"
}
