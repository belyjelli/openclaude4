package providers

import (
	"fmt"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
)

// NewStreamClient builds the chat stream client for the configured provider.
func NewStreamClient() (core.StreamClient, error) {
	switch config.ProviderName() {
	case "ollama":
		return openaicomp.NewOllama()
	default:
		name := config.ProviderName()
		if name != "" && name != "openai" {
			return nil, fmt.Errorf("unknown provider %q (try openai or ollama)", name)
		}
		return openaicomp.New()
	}
}

// StreamClientInfo is implemented by *openaicomp.Client for CLI banners.
type StreamClientInfo interface {
	core.StreamClient
	Model() string
	BaseURL() string
	RedactedAPIKeySummary() string
	ProviderKind() string
}

// AsStreamClientInfo narrows to banner/doctor helpers when the implementation supports it.
func AsStreamClientInfo(c core.StreamClient) (StreamClientInfo, bool) {
	si, ok := c.(StreamClientInfo)
	return si, ok
}

// PingProviderBestEffort returns a one-line reachability hint (no fatal on failure).
func PingProviderBestEffort() string {
	switch config.ProviderName() {
	case "ollama":
		return pingOllama()
	default:
		return pingOpenAI()
	}
}

func pingOllama() string {
	base := strings.TrimSuffix(config.OllamaChatBase(), "/v1")
	u := base + "/api/tags"
	return httpGetLine(u)
}

func pingOpenAI() string {
	if config.APIKey() == "" {
		return "OpenAI-compatible: no API key configured"
	}
	u := config.BaseURL()
	if u == "" {
		return "OpenAI-compatible: using default api.openai.com (key set)"
	}
	return "OpenAI-compatible: custom base URL configured (key set)"
}

func httpGetLine(url string) string {
	return pingHTTP(url)
}
