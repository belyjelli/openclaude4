package config

import (
	"strings"

	"github.com/spf13/viper"
)

// InitTUIDefaults sets viper defaults for TUI presentation keys.
func InitTUIDefaults() {
	viper.SetDefault("tui.busy_verbose_tokens", false)
}

// TUISpinnerVerbConfig returns optional user spinner verbs (openclaude3-style).
// When replace is true and verbs is non-empty, only those verbs are used.
// When replace is false and verbs is non-empty, they are appended to built-ins.
// YAML / JSON example:
//
//	tui:
//	  spinner_verbs:
//	    mode: append   # or replace
//	    verbs: ["Shippifying", "Rubberducking"]
func TUISpinnerVerbConfig() (replace bool, verbs []string) {
	mode := strings.ToLower(strings.TrimSpace(viper.GetString("tui.spinner_verbs.mode")))
	verbs = viper.GetStringSlice("tui.spinner_verbs.verbs")
	return mode == "replace", verbs
}

// TUIBusyLineVerboseTokens mirrors v3 verbose: always show rough transcript token count on the busy line.
// Config: tui.busy_verbose_tokens; env OPENCLAUDE_TUI_BUSY_VERBOSE_TOKENS (see bindViperEnv).
func TUIBusyLineVerboseTokens() bool {
	return viper.GetBool("tui.busy_verbose_tokens")
}
