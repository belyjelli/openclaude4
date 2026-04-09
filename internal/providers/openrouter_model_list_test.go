package providers

import (
	"testing"

	"github.com/OpenRouterTeam/go-sdk/models/components"
)

func TestSkipOpenRouterModelForChat(t *testing.T) {
	if !skipOpenRouterModelForChat("openai/text-embedding-3-small", nil) {
		t.Fatal("expected embedding id skipped")
	}
	if skipOpenRouterModelForChat("openai/gpt-4o", nil) {
		t.Fatal("expected chat id kept")
	}
	mod := "embedding"
	m := &components.Model{Architecture: components.ModelArchitecture{Modality: &mod}}
	if !skipOpenRouterModelForChat("x/y", m) {
		t.Fatal("expected embedding modality skipped")
	}
}
