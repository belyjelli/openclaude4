package openaicomp

import (
	"context"
	"errors"

	"github.com/gitlawb/openclaude4/internal/config"
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
	kind   string // "openai" | "ollama"
}

// New builds an OpenAI or OpenAI-compatible remote client (requires OPENAI_API_KEY).
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
		kind:   "openai",
	}, nil
}

// NewOllama uses the local Ollama OpenAI-compatible endpoint (no API key).
func NewOllama() (*Client, error) {
	base := config.OllamaChatBase()
	cfg := sdk.DefaultConfig("ollama")
	cfg.BaseURL = base
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.OllamaModel(),
		apiKey: "",
		base:   base,
		kind:   "ollama",
	}, nil
}

// ProviderKind returns "openai" or "ollama".
func (c *Client) ProviderKind() string {
	if c.kind != "" {
		return c.kind
	}
	return "openai"
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

// Model returns the configured model name.
func (c *Client) Model() string { return c.model }

// BaseURL returns the configured base URL override (empty means SDK default).
func (c *Client) BaseURL() string { return c.base }

// RedactedAPIKeySummary returns a short redacted form for display (never the full secret).
func (c *Client) RedactedAPIKeySummary() string {
	if c.ProviderKind() == "ollama" {
		return "(local Ollama — no API key)"
	}
	k := c.apiKey
	if len(k) <= 8 {
		return "(set)"
	}
	return k[:4] + "…" + k[len(k)-4:]
}
