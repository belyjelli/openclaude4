package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoadAgentProfiles_dedupe(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	_ = viper.MergeConfigMap(map[string]any{
		"agents": []any{
			map[string]any{"type": "A", "instructions": "first"},
			map[string]any{"type": "a", "instructions": "second"},
			map[string]any{"type": "", "instructions": "skip"},
		},
	})
	got := LoadAgentProfiles()
	if len(got) != 1 || got[0].Type != "A" || got[0].Instructions != "first" {
		t.Fatalf("got %#v", got)
	}
}
