package startupbanner

import "testing"

func TestSplashDisabled(t *testing.T) {
	t.Setenv("OPENCLAUDE_NO_SPLASH", "")
	if SplashDisabled() {
		t.Fatal("expected splash enabled when env unset")
	}
	t.Setenv("OPENCLAUDE_NO_SPLASH", "1")
	if !SplashDisabled() {
		t.Fatal("expected splash disabled for OPENCLAUDE_NO_SPLASH=1")
	}
	t.Setenv("OPENCLAUDE_NO_SPLASH", "TRUE")
	if !SplashDisabled() {
		t.Fatal("expected splash disabled for OPENCLAUDE_NO_SPLASH=TRUE")
	}
}

func TestProviderLabel(t *testing.T) {
	t.Parallel()
	name, local := providerLabel("ollama", "http://127.0.0.1:11434/v1", "llama3")
	if name != "Ollama" || !local {
		t.Fatalf("ollama: got %q local=%v", name, local)
	}
	name, local = providerLabel("openai", "https://api.openai.com/v1", "gpt-4o")
	if name != "OpenAI" || local {
		t.Fatalf("openai cloud: got %q local=%v", name, local)
	}
	name, local = providerLabel("openai", "http://localhost:8080/v1", "m")
	if name != "Local (OpenAI-compatible)" || !local {
		t.Fatalf("local openai-compat: got %q local=%v", name, local)
	}
	name, local = providerLabel("openai", "https://api.openai.com/v1", "deepseek-chat")
	if name != "DeepSeek" || local {
		t.Fatalf("deepseek by model: got %q local=%v", name, local)
	}
	name, local = providerLabel("openai", "https://api.fireworks.ai/v1", "qwen")
	if name != "Fireworks AI" || local {
		t.Fatalf("fireworks: got %q local=%v", name, local)
	}
	name, local = providerLabel("openai", "https://api.x.ai/v1", "grok")
	if name != "xAI" || local {
		t.Fatalf("xai: got %q local=%v", name, local)
	}
}
