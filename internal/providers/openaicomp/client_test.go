package openaicomp

import (
	"errors"
	"testing"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/spf13/viper"
)

func TestNew_OpenRouterBaseWithoutKeys(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "https://openrouter.ai/api/v1")
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	_, err := New()
	if !errors.Is(err, ErrMissingOpenRouterOrOpenAIKey) {
		t.Fatalf("New() err = %v, want %v", err, ErrMissingOpenRouterOrOpenAIKey)
	}
}

func TestNew_OpenRouterBaseWithOpenRouterKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_KEY", "sk-or-test-key-12345")
	t.Setenv("OPENAI_BASE_URL", "https://openrouter.ai/api/v1")
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.ProviderKind() != "openai" {
		t.Fatalf("client = %v kind = %s", c, c.ProviderKind())
	}
}

func TestNewOpenRouter_MissingKey(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENCLAUDE_PROVIDER", "openrouter")
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	_, err := NewOpenRouter()
	if !errors.Is(err, ErrMissingOpenRouterKey) {
		t.Fatalf("NewOpenRouter() err = %v", err)
	}
}

func TestNewOpenRouter_WithKey(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "sk-or-test-key-12345")
	t.Setenv("OPENCLAUDE_PROVIDER", "openrouter")
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	c, err := NewOpenRouter()
	if err != nil {
		t.Fatal(err)
	}
	if c.ProviderKind() != "openrouter" {
		t.Fatalf("kind = %s", c.ProviderKind())
	}
	if c.BaseURL() != "https://openrouter.ai/api/v1" {
		t.Fatalf("base = %q", c.BaseURL())
	}
}
