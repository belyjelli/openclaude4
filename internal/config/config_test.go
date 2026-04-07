package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestModel_DefaultWhenUnset(t *testing.T) {
	viper.Reset()
	Load()
	if got := Model(); got != defaultModel {
		t.Fatalf("Model() = %q, want %q", got, defaultModel)
	}
}
