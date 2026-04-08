package providers

import (
	"context"
	"errors"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	sdk "github.com/sashabaranov/go-openai"
)

// doctorBannerClient implements [core.StreamClient] and [StreamClientInfo] from loaded config only,
// so the CLI splash can render when [NewStreamClient] fails (e.g. missing API key).
type doctorBannerClient struct{}

// DoctorBannerClient returns a minimal client used for banner/doctor display when no chat client can be built.
func DoctorBannerClient() core.StreamClient {
	return doctorBannerClient{}
}

func (doctorBannerClient) StreamChatWithTools(context.Context, []sdk.ChatCompletionMessage, []sdk.Tool) (*sdk.ChatCompletionStream, error) {
	return nil, errors.New("doctor banner: not a chat client")
}

func (doctorBannerClient) Model() string {
	return config.Model()
}

func (doctorBannerClient) ProviderKind() string {
	return config.ProviderName()
}

func (doctorBannerClient) BaseURL() string {
	switch strings.ToLower(strings.TrimSpace(config.ProviderName())) {
	case "ollama":
		return config.OllamaChatBase()
	case "gemini":
		return config.GeminiBaseURL()
	case "github":
		return config.GitHubModelsBaseURL()
	default:
		return config.BaseURL()
	}
}

func (doctorBannerClient) RedactedAPIKeySummary() string {
	if config.APIKey() != "" {
		return "set"
	}
	return "not set"
}
