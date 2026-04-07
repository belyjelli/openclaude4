package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestPrecedence_EnvOverYAMLFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	yaml := "provider:\n  name: openai\n  model: from-yaml\n"
	if err := os.WriteFile(filepath.Join(dir, "openclaude.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENCLAUDE_PROVIDER", "ollama")
	t.Setenv("OLLAMA_MODEL", "from-env")

	viper.Reset()
	Load("")

	if got := ProviderName(); got != "ollama" {
		t.Fatalf("ProviderName() = %q, want ollama", got)
	}
	if got := Model(); got != "from-env" {
		t.Fatalf("Model() = %q, want from-env", got)
	}
}

func TestPrecedence_YAMLFileOverV3Profile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	v3 := `{"profile":"openai","env":{}}`
	if err := os.WriteFile(filepath.Join(dir, ".openclaude-profile.json"), []byte(v3), 0o600); err != nil {
		t.Fatal(err)
	}
	yaml := "provider:\n  name: ollama\n  model: llama-from-yaml\n"
	if err := os.WriteFile(filepath.Join(dir, "openclaude.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("OPENCLAUDE_PROVIDER")
		_ = os.Unsetenv("OLLAMA_MODEL")
		_ = os.Unsetenv("OPENAI_MODEL")
	})

	viper.Reset()
	Load("")

	if got := ProviderName(); got != "ollama" {
		t.Fatalf("ProviderName() = %q, want ollama", got)
	}
	if got := Model(); got != "llama-from-yaml" {
		t.Fatalf("Model() = %q, want llama-from-yaml", got)
	}
}

func TestValidate_UnknownProvider(t *testing.T) {
	viper.Reset()
	Load("")
	viper.Set("provider.name", "not-a-real-provider")
	if err := Validate(); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestValidate_OKForDefault(t *testing.T) {
	viper.Reset()
	Load("")
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}
