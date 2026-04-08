package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// MergedV3ProfilePath is the path of the v3 profile JSON merged into viper by the last
// [MergeV3Profile] call, or empty if none was merged.
var MergedV3ProfilePath string

// v3ProfileJSON matches OpenClaude v3 [.openclaude-profile.json](https://github.com/Gitlawb/openclaude).
type v3ProfileJSON struct {
	Profile string         `json:"profile"`
	Env     map[string]any `json:"env"`
}

// MergeV3Profile loads the first existing v3 profile and merges it into viper (lowest priority:
// overridden by openclaude.yaml and by environment variables). Search order: cwd, then home.
func MergeV3Profile(cwd, home string) {
	MergedV3ProfilePath = ""
	const name = ".openclaude-profile.json"
	paths := make([]string, 0, 2)
	if cwd != "" {
		paths = append(paths, filepath.Join(cwd, name))
	}
	if home != "" {
		h := filepath.Join(home, name)
		if len(paths) == 0 || paths[0] != h {
			paths = append(paths, h)
		}
	}
	for _, p := range paths {
		if mergeOneV3Profile(p) {
			return
		}
	}
}

func mergeOneV3Profile(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc v3ProfileJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "openclaude: %s: invalid JSON: %v\n", path, err)
		return false
	}
	if doc.Profile == "" || doc.Env == nil {
		return false
	}

	m := map[string]any{}

	switch strings.ToLower(strings.TrimSpace(doc.Profile)) {
	case "openai":
		m["provider"] = map[string]any{"name": "openai"}
	case "ollama":
		m["provider"] = map[string]any{"name": "ollama"}
	case "gemini":
		m["provider"] = map[string]any{"name": "gemini"}
	case "codex":
		m["provider"] = map[string]any{"name": "codex"}
	case "atomic-chat":
		m["provider"] = map[string]any{"name": "openai"}
	default:
		return false
	}

	openai := map[string]any{}
	gemini := map[string]any{}
	ollama := map[string]any{}
	provider := map[string]any{}

	put := func(dst map[string]any, key, val string) {
		if val != "" {
			dst[key] = val
		}
	}
	env := doc.Env
	if s := envStr(env, "OPENAI_API_KEY"); s != "" {
		put(openai, "api_key", s)
	}
	if s := envStr(env, "OPENAI_BASE_URL"); s != "" {
		put(provider, "base_url", s)
	}
	if s := envStr(env, "OPENAI_MODEL"); s != "" {
		put(provider, "model", s)
	}
	if s := envStr(env, "GEMINI_API_KEY"); s != "" {
		put(gemini, "api_key", s)
	} else if s := envStr(env, "GOOGLE_API_KEY"); s != "" {
		put(gemini, "api_key", s)
	}
	if s := envStr(env, "GEMINI_MODEL"); s != "" {
		put(gemini, "model", s)
	}
	if s := envStr(env, "GEMINI_BASE_URL"); s != "" {
		put(gemini, "base_url", s)
	}
	if s := envStr(env, "OLLAMA_HOST"); s != "" {
		put(ollama, "host", s)
	}
	if s := envStr(env, "OLLAMA_MODEL"); s != "" {
		put(ollama, "model", s)
	}

	if len(openai) > 0 {
		m["openai"] = openai
	}
	if len(gemini) > 0 {
		m["gemini"] = gemini
	}
	if len(ollama) > 0 {
		m["ollama"] = ollama
	}
	if len(provider) > 0 {
		mergeProviderMap(m, provider)
	}

	if err := viper.MergeConfigMap(m); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "openclaude: profile merge: %v\n", err)
		return false
	}
	MergedV3ProfilePath = path
	return true
}

func mergeProviderMap(m map[string]any, patch map[string]any) {
	cur, ok := m["provider"].(map[string]any)
	if !ok {
		m["provider"] = patch
		return
	}
	for k, v := range patch {
		cur[k] = v
	}
	m["provider"] = cur
}

func envStr(env map[string]any, key string) string {
	v, ok := env[key]
	if !ok {
		return ""
	}
	s, ok := asEnvString(v)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func asEnvString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t), true
	case json.Number:
		s := t.String()
		return s, s != ""
	case float64:
		return fmt.Sprintf("%.0f", t), true
	case bool:
		return fmt.Sprint(t), true
	default:
		return "", false
	}
}
