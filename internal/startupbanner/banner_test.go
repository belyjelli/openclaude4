package startupbanner

import (
	"strings"
	"testing"
)

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

func TestTUIBannerContent_plainPointsAtSubtitle(t *testing.T) {
	t.Parallel()
	s := TUIBannerContent("9.9.9", "MCP: 1 tool", false, "")
	if !strings.Contains(s, "subtitle") {
		t.Fatalf("expected plain TUI banner to mention subtitle, got %q", s)
	}
	if strings.Contains(s, "Provider:") {
		t.Fatalf("plain TUI banner should not mimic frozen provider card: %q", s)
	}
}

func TestTUIBannerContent_ansiHasLiveHintNotProviderRows(t *testing.T) {
	t.Parallel()
	s := TUIBannerContent("1.0.0", "", true, "")
	if !strings.Contains(s, "subtitle") {
		t.Fatalf("expected ANSI TUI banner to mention subtitle: %q", s)
	}
	// Full Render() uses a "Provider" box-row key; TUI splash must not duplicate that card.
	if strings.Contains(s, "Provider") {
		t.Fatalf("TUI ANSI banner should not contain Provider row (stale after /provider): %q", s)
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
