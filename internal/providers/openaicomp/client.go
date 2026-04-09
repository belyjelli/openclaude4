package openaicomp

import (
	"context"
	"errors"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	sdk "github.com/sashabaranov/go-openai"
)

// ErrMissingAPIKey is returned when OPENAI_API_KEY is unset.
var ErrMissingAPIKey = errors.New("OPENAI_API_KEY is not set")

// ErrMissingOpenRouterKey is returned when OPENROUTER_KEY is unset for provider openrouter or OpenRouter chat.
var ErrMissingOpenRouterKey = errors.New("OPENROUTER_KEY or OPENROUTER_API_KEY is not set")

// ErrMissingOpenRouterOrOpenAIKey is returned when the base URL targets OpenRouter but neither OPENAI_API_KEY nor OPENROUTER_KEY is set.
var ErrMissingOpenRouterOrOpenAIKey = errors.New("set OPENAI_API_KEY or OPENROUTER_KEY when using OpenRouter (OPENAI_BASE_URL contains openrouter.ai)")

// ErrMissingGeminiKey is returned when GEMINI_API_KEY / GOOGLE_API_KEY is unset.
var ErrMissingGeminiKey = errors.New("GEMINI_API_KEY or GOOGLE_API_KEY is not set")

// Client wraps the OpenAI-compatible HTTP client.
type Client struct {
	inner  *sdk.Client
	model  string
	apiKey string
	base   string
	kind   string // "openai" | "ollama" | "gemini" | "github" | "openrouter"
}

// New builds an OpenAI or OpenAI-compatible remote client (uses OPENAI_API_KEY, or OPENROUTER_KEY when base targets OpenRouter).
func New() (*Client, error) {
	base := config.BaseURL()
	key := config.EffectiveOpenAICompatAPIKey()
	if key == "" {
		if config.BaseURLLooksLikeOpenRouter(base) {
			return nil, ErrMissingOpenRouterOrOpenAIKey
		}
		return nil, ErrMissingAPIKey
	}
	cfg := sdk.DefaultConfig(key)
	if base != "" {
		cfg.BaseURL = base
	}
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.Model(),
		apiKey: key,
		base:   base,
		kind:   "openai",
	}, nil
}

// NewOpenRouter uses OpenRouter's OpenAI-compatible endpoint (OPENROUTER_KEY / openrouter.api_key).
func NewOpenRouter() (*Client, error) {
	key := config.OpenRouterAPIKey()
	if key == "" {
		return nil, ErrMissingOpenRouterKey
	}
	base := config.OpenRouterChatBase()
	cfg := sdk.DefaultConfig(key)
	cfg.BaseURL = base
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.OpenRouterModel(),
		apiKey: key,
		base:   base,
		kind:   "openrouter",
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

// NewGemini uses Google's OpenAI-compatible Gemini endpoint.
func NewGemini() (*Client, error) {
	key := config.GeminiAPIKey()
	if key == "" {
		return nil, ErrMissingGeminiKey
	}
	base := config.GeminiBaseURL()
	cfg := sdk.DefaultConfig(key)
	cfg.BaseURL = base
	return &Client{
		inner:  sdk.NewClientWithConfig(cfg),
		model:  config.GeminiModel(),
		apiKey: key,
		base:   base,
		kind:   "gemini",
	}, nil
}

// ProviderKind returns "openai", "ollama", "gemini", "github", or "openrouter".
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

// WithModel returns a shallow copy of the client using a different model id (same credentials and base URL).
func (c *Client) WithModel(model string) *Client {
	if c == nil {
		return nil
	}
	m := strings.TrimSpace(model)
	if m == "" {
		return c
	}
	cp := *c
	cp.model = m
	return &cp
}

// Model returns the configured model name.
func (c *Client) Model() string { return c.model }

// BaseURL returns the configured base URL override (empty means SDK default).
func (c *Client) BaseURL() string { return c.base }

// RedactedAPIKeySummary returns a short redacted form for display (never the full secret).
func (c *Client) RedactedAPIKeySummary() string {
	switch c.ProviderKind() {
	case "ollama":
		return "(local Ollama — no API key)"
	case "github":
		if len(c.apiKey) <= 8 {
			return "(set)"
		}
		return c.apiKey[:4] + "…" + c.apiKey[len(c.apiKey)-4:]
	case "gemini":
		if len(c.apiKey) <= 8 {
			return "(set)"
		}
		return c.apiKey[:4] + "…" + c.apiKey[len(c.apiKey)-4:]
	case "openrouter":
		if len(c.apiKey) <= 8 {
			return "(set)"
		}
		return c.apiKey[:4] + "…" + c.apiKey[len(c.apiKey)-4:]
	}
	k := c.apiKey
	if len(k) <= 8 {
		return "(set)"
	}
	return k[:4] + "…" + k[len(k)-4:]
}
