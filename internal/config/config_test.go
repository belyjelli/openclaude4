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

func TestMCPServers_UnmarshalKey(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("mcp", map[string]any{
		"servers": []any{
			map[string]any{
				"name":     "demo",
				"command":  []any{"node", "srv.js"},
				"approval": "never",
			},
			map[string]any{"name": "", "command": []any{"x"}}, // skipped: empty name
		},
	})
	list := MCPServers()
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(list), list)
	}
	if list[0].Name != "demo" || len(list[0].Command) != 2 || list[0].Command[0] != "node" {
		t.Fatalf("%+v", list[0])
	}
	if list[0].Approval != "never" {
		t.Fatalf("approval = %q", list[0].Approval)
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

// OPENAI_MODEL must not override the model when a non-openai provider is selected (stale shells / v3 habits).
func TestModel_OllamaIgnoresOPENAI_MODEL(t *testing.T) {
	t.Setenv("OPENCLAUDE_PROVIDER", "ollama")
	t.Setenv("OPENAI_MODEL", "gpt-4o")
	t.Setenv("OLLAMA_MODEL", "")
	viper.Reset()
	Load("")
	if got := Model(); got != defaultOllamaModel {
		t.Fatalf("Model() = %q, want default Ollama %q", got, defaultOllamaModel)
	}
}

func TestModel_OpenAIUsesOPENAI_MODEL(t *testing.T) {
	t.Setenv("OPENCLAUDE_PROVIDER", "openai")
	t.Setenv("OPENAI_MODEL", "gpt-4o")
	viper.Reset()
	Load("")
	if got := Model(); got != "gpt-4o" {
		t.Fatalf("Model() = %q, want gpt-4o", got)
	}
}
