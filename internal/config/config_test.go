package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestModel_DefaultWhenUnset(t *testing.T) {
	viper.Reset()
	Load("")
	if got := Model(); got != defaultModel {
		t.Fatalf("Model() = %q, want %q", got, defaultModel)
	}
}

func TestOllamaChatBase_Default(t *testing.T) {
	viper.Reset()
	Load("")
	if got := OllamaChatBase(); got != "http://127.0.0.1:11434/v1" {
		t.Fatalf("OllamaChatBase() = %q", got)
	}
}

func TestProviderName_OllamaFromEnv(t *testing.T) {
	t.Setenv("OPENCLAUDE_PROVIDER", "ollama")
	t.Setenv("OLLAMA_MODEL", "mistral")
	viper.Reset()
	Load("")
	if ProviderName() != "ollama" {
		t.Fatalf("ProviderName() = %q", ProviderName())
	}
	if Model() != "mistral" {
		t.Fatalf("Model() = %q, want mistral", Model())
	}
}
