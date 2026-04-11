package providerwizard

import (
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/spf13/viper"
)

func TestApplyToViper_NotFinished(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	w := New()
	if err := w.ApplyToViper(); err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyToViper_Cancelled(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	w := New()
	w.Cancel()
	if err := w.ApplyToViper(); err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyToViper_OpenAI(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("OPENAI_API_KEY", "")
	config.Load("")
	w := New()
	_ = w.SelectMenuIndex(0)
	_ = w.SelectMenuIndex(len(w.MenuOptions()) - 1) // custom base URL
	_ = w.SubmitText("https://api.example/v1")
	_ = w.SubmitText("gpt-custom")
	if err := w.ApplyToViper(); err != nil {
		t.Fatal(err)
	}
	if config.ProviderName() != "openai" {
		t.Fatalf("provider: %q", config.ProviderName())
	}
	if config.Model() != "gpt-custom" {
		t.Fatalf("model: %q", config.Model())
	}
	if config.BaseURL() != "https://api.example/v1" {
		t.Fatalf("base: %q", config.BaseURL())
	}
}

func TestApplyToViper_OpenAI_ClearBase(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("OPENAI_API_KEY", "")
	config.Load("")
	viper.Set("provider.base_url", "https://old.example/v1")
	w := New()
	_ = w.SelectMenuIndex(0)
	_ = w.SelectMenuIndex(0) // default official base
	_ = w.SubmitText("m1")
	if err := w.ApplyToViper(); err != nil {
		t.Fatal(err)
	}
	if config.BaseURL() != "" {
		t.Fatalf("want empty base, got %q", config.BaseURL())
	}
}

func TestApplyToViper_Ollama(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.Load("")
	w := New()
	_ = w.SelectMenuIndex(1)
	_ = w.SubmitText("http://127.0.0.1:11434")
	_ = w.SubmitText("mistral")
	if err := w.ApplyToViper(); err != nil {
		t.Fatal(err)
	}
	if config.ProviderName() != "ollama" {
		t.Fatal(config.ProviderName())
	}
	if !strings.Contains(config.OllamaChatBase(), "127.0.0.1:11434") {
		t.Fatal(config.OllamaChatBase())
	}
	if config.OllamaModel() != "mistral" {
		t.Fatal(config.OllamaModel())
	}
}
