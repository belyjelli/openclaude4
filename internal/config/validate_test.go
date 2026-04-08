package config

import (
	"errors"
	"testing"

	"github.com/gitlawb/openclaude4/internal/providererrs"
	"github.com/spf13/viper"
)

func TestValidate_CodexReturnsNotImplemented(t *testing.T) {
	viper.Reset()
	Load("")
	viper.Set("provider.name", "codex")
	err := Validate()
	if err == nil {
		t.Fatal("expected error for codex provider")
	}
	if !errors.Is(err, providererrs.ErrCodexNotImplemented) {
		t.Fatalf("Validate() = %v, want errors.Is(ErrCodexNotImplemented)", err)
	}
}
