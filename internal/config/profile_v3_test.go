package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestMergeV3Profile_Gemini(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	prof := filepath.Join(tmp, ".openclaude-profile.json")
	raw := `{"profile":"gemini","env":{"GEMINI_API_KEY":"AIzaSyDummyKeyForTestOnly","GEMINI_MODEL":"gemini-2.0-flash"}}`
	if err := os.WriteFile(prof, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	bindViperEnv()
	MergeV3Profile(tmp, "")
	if MergedV3ProfilePath != prof {
		t.Fatalf("MergedV3ProfilePath = %q want %q", MergedV3ProfilePath, prof)
	}

	if ProviderName() != "gemini" {
		t.Fatalf("ProviderName = %q", ProviderName())
	}
	if GeminiAPIKey() == "" {
		t.Fatal("expected GEMINI_API_KEY from profile")
	}
	if GeminiModel() != "gemini-2.0-flash" {
		t.Fatalf("GeminiModel = %q", GeminiModel())
	}
}

func TestMergeV3Profile_YamlOverridesProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.WriteFile(filepath.Join(tmp, ".openclaude-profile.json"), []byte(`{"profile":"ollama","env":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "openclaude.yaml"), []byte("provider:\n  name: openai\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	Load("")
	if got := filepath.Clean(MergedV3ProfilePath); got != filepath.Clean(filepath.Join(tmp, ".openclaude-profile.json")) {
		t.Fatalf("MergedV3ProfilePath = %q", MergedV3ProfilePath)
	}
	if ProviderName() != "openai" {
		t.Fatalf("want openai from yaml over profile ollama, got %q", ProviderName())
	}
}
