package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestBaseURLLooksLikeOpenRouter(t *testing.T) {
	if !BaseURLLooksLikeOpenRouter("https://openrouter.ai/api/v1") {
		t.Fatal("expected true")
	}
	if BaseURLLooksLikeOpenRouter("https://api.openai.com/v1") {
		t.Fatal("expected false")
	}
}

func TestEffectiveOpenAICompatAPIKey_OpenRouterBasePrefersOpenRouterKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "https://openrouter.ai/api/v1")
	t.Setenv("OPENROUTER_KEY", "sk-or-test-key")
	viper.Reset()
	t.Cleanup(viper.Reset)
	Load("")
	if got := EffectiveOpenAICompatAPIKey(); got != "sk-or-test-key" {
		t.Fatalf("EffectiveOpenAICompatAPIKey() = %q", got)
	}
}

func TestEffectiveOpenAICompatAPIKey_OpenAIKeyWinsOverOpenRouter(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-openai")
	t.Setenv("OPENAI_BASE_URL", "https://openrouter.ai/api/v1")
	t.Setenv("OPENROUTER_KEY", "sk-or-other")
	viper.Reset()
	t.Cleanup(viper.Reset)
	Load("")
	if got := EffectiveOpenAICompatAPIKey(); got != "sk-openai" {
		t.Fatalf("EffectiveOpenAICompatAPIKey() = %q", got)
	}
}

func TestEffectiveOpenAICompatAPIKey_NonOpenRouterBaseIgnoresOpenRouterKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "https://api.example.com/v1")
	t.Setenv("OPENROUTER_KEY", "sk-or-test")
	viper.Reset()
	t.Cleanup(viper.Reset)
	Load("")
	if got := EffectiveOpenAICompatAPIKey(); got != "" {
		t.Fatalf("EffectiveOpenAICompatAPIKey() = %q, want empty", got)
	}
}

func TestOpenRouterModel_Default(t *testing.T) {
	t.Setenv("OPENCLAUDE_PROVIDER", "openrouter")
	viper.Reset()
	t.Cleanup(viper.Reset)
	Load("")
	if got := OpenRouterModel(); got != "openai/gpt-4o-mini" {
		t.Fatalf("OpenRouterModel() = %q", got)
	}
}

func TestOpenRouterChatBase_Default(t *testing.T) {
	t.Setenv("OPENCLAUDE_PROVIDER", "openrouter")
	t.Setenv("OPENAI_BASE_URL", "")
	viper.Reset()
	t.Cleanup(viper.Reset)
	Load("")
	want := "https://openrouter.ai/api/v1"
	if got := OpenRouterChatBase(); got != want {
		t.Fatalf("OpenRouterChatBase() = %q, want %q", got, want)
	}
}
