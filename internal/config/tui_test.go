package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestTUISpinnerVerbConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("tui.spinner_verbs.mode", "append")
	viper.Set("tui.spinner_verbs.verbs", []string{"Alpha", "Beta"})
	replace, verbs := TUISpinnerVerbConfig()
	if replace {
		t.Fatal("expected append mode")
	}
	if len(verbs) != 2 || verbs[0] != "Alpha" {
		t.Fatalf("verbs: %v", verbs)
	}
}

func TestTUISpinnerVerbConfigReplace(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("tui.spinner_verbs.mode", "replace")
	viper.Set("tui.spinner_verbs.verbs", []string{"Only"})
	replace, verbs := TUISpinnerVerbConfig()
	if !replace || len(verbs) != 1 || verbs[0] != "Only" {
		t.Fatalf("replace=%v verbs=%v", replace, verbs)
	}
}

func TestTUIBusyLineVerboseTokens(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	if TUIBusyLineVerboseTokens() {
		t.Fatal("default off")
	}
	viper.Set("tui.busy_verbose_tokens", true)
	if !TUIBusyLineVerboseTokens() {
		t.Fatal("expected on")
	}
}
