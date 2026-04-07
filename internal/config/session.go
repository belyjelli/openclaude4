package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// SessionDir returns the directory for on-disk session JSON files.
func SessionDir(home string) string {
	if v := strings.TrimSpace(viper.GetString("session.dir")); v != "" {
		return expandPath(v, home)
	}
	if home == "" {
		return filepath.Join(".openclaude", "sessions")
	}
	return filepath.Join(home, ".local", "share", "openclaude", "sessions")
}

func expandPath(p, home string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~/") && home != "" {
		return filepath.Join(home, p[2:])
	}
	if p == "~" && home != "" {
		return home
	}
	return p
}

// SessionDisabled is true when --no-session / session.disabled suppresses persistence.
func SessionDisabled() bool {
	if viper.GetBool("session.disabled") {
		return true
	}
	v := strings.TrimSpace(os.Getenv("OPENCLAUDE_NO_SESSION"))
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// SessionCompactTokenThreshold enables automatic compaction/summarization when rough
// token estimate exceeds this value. 0 means off.
func SessionCompactTokenThreshold() int {
	return viper.GetInt("session.compact_token_threshold")
}

// SessionSummarizeOverThreshold requests an LLM summary instead of lossy tail compaction
// when the token threshold trips (falls back to compaction if the call fails).
func SessionSummarizeOverThreshold() bool {
	return viper.GetBool("session.summarize_over_threshold")
}

// SessionCompactKeepMessages is the tail size for lossy compaction (system + last N).
func SessionCompactKeepMessages() int {
	v := viper.GetInt("session.compact_keep_messages")
	if v <= 0 {
		return 24
	}
	return v
}

// InitSessionDefaults sets viper defaults for session keys.
func InitSessionDefaults() {
	viper.SetDefault("session.disabled", false)
	viper.SetDefault("session.compact_token_threshold", 0)
	viper.SetDefault("session.summarize_over_threshold", false)
	viper.SetDefault("session.compact_keep_messages", 24)
}

// EffectiveSessionDir resolves the session directory (absolute when possible).
func EffectiveSessionDir() string {
	home, _ := os.UserHomeDir()
	d := SessionDir(home)
	if filepath.IsAbs(d) {
		return d
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Clean(filepath.Join(cwd, d))
	}
	return d
}
