package openaicomp

import (
	"context"
	"errors"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	sdk "github.com/sashabaranov/go-openai"
)

// ErrMissingGitHubToken is returned when GitHub token is unset.
var ErrMissingGitHubToken = errors.New("GITHUB_TOKEN or GITHUB_PAT is not set")

// NewGitHubModels uses GitHub Models API (OpenAI-compatible endpoint).
// GitHub Models provides access to various models via github.com models.
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

// StreamChat starts a streaming chat completion (no tools).
func (c *Client) StreamChat(ctx context.Context, messages []sdk.ChatCompletionMessage) (*sdk.ChatCompletionStream, error) {
	return c.inner.CreateChatCompletionStream(ctx, sdk.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	})
}

// StreamChatWithTools streams one assistant turn with function-calling enabled.
func (c *Client) StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error) {
	return c.inner.CreateChatCompletionStream(ctx, sdk.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    toolList,
		Stream:   true,
	})
}

// ProviderKind returns "github" for GitHub Models.
func (c *Client) ProviderKind() string {
	if c.kind != "" {
		return c.kind
	}
	return "github"
}

// Model returns the configured model name.
func (c *Client) Model() string { return c.model }

// BaseURL returns the configured base URL override (empty means SDK default).
func (c *Client) BaseURL() string { return c.base }

// RedactedAPIKeySummary returns a short redacted form for display (never the full secret).
func (c *Client) RedactedAPIKeySummary() string {
	k := c.apiKey
	if len(k) <= 8 {
		return "(set)"
	}
	return k[:4] + "…" + k[len(k)-4:]
}
