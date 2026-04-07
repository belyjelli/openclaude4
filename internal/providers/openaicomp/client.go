package openaicomp

import (
	"context"
	"errors"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/providers"
	sdk "github.com/sashabaranov/go-openai"
)

// ErrMissingAPIKey is returned when OPENAI_API_KEY is unset.
var ErrMissingAPIKey = errors.New("OPENAI_API_KEY is not set")

// Client wraps the OpenAI-compatible HTTP client.
type Client struct {
	inner  *sdk.Client
	model  string
	apiKey string
	base   string
}

var _ providers.ChatStreamer = (*Client)(nil)

// New builds a client from global config (env / viper).
func New() (*Client, error) {
	key := config.APIKey()
	if key == "" {
		return nil, ErrMissingAPIKey
	}
	cfg := sdk.DefaultConfig(key)
	if base := config.BaseURL(); base != "" {
		cfg.BaseURL = base
	}
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.Model(),
		apiKey: key,
		base:   config.BaseURL(),
	}, nil
}

// StreamChat starts a streaming chat completion.
func (c *Client) StreamChat(ctx context.Context, messages []sdk.ChatCompletionMessage) (*sdk.ChatCompletionStream, error) {
	return c.inner.CreateChatCompletionStream(ctx, sdk.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	})
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
