package openaicomp

import (
	"errors"

	"github.com/gitlawb/openclaude4/internal/config"
	sdk "github.com/sashabaranov/go-openai"
)

// ErrMissingGitHubToken is returned when GitHub token is unset.
var ErrMissingGitHubToken = errors.New("GITHUB_TOKEN or GITHUB_PAT is not set")

// NewGitHubModels uses GitHub Models API (OpenAI-compatible endpoint).
func NewGitHubModels() (*Client, error) {
	key := config.GitHubToken()
	if key == "" {
		return nil, ErrMissingGitHubToken
	}
	base := config.GitHubModelsBaseURL()
	cfg := sdk.DefaultConfig(key)
	cfg.BaseURL = base
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.GitHubModelsModel(),
		apiKey: key,
		base:   base,
		kind:   "github",
	}, nil
}
