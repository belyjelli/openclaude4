package config

import (
	"strings"

	"github.com/spf13/viper"
)

// AgentProfile is one entry under YAML key `agents:` (v3-style @agent-<type> / @"type (agent)" hints).
// Type matches v3 token grammar: [\w:.@-]+ after optional `agent-` prefix in prompts.
type AgentProfile struct {
	Type         string `mapstructure:"type"`
	Instructions string `mapstructure:"instructions"`
}

// LoadAgentProfiles reads optional `agents:` from merged viper config (trimmed, deduped by lowercase type).
func LoadAgentProfiles() []AgentProfile {
	if !viper.IsSet("agents") {
		return nil
	}
	var raw []AgentProfile
	if err := viper.UnmarshalKey("agents", &raw); err != nil || len(raw) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []AgentProfile
	for _, a := range raw {
		t := strings.TrimSpace(a.Type)
		if t == "" {
			continue
		}
		low := strings.ToLower(t)
		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		out = append(out, AgentProfile{
			Type:         t,
			Instructions: strings.TrimSpace(a.Instructions),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// AgentProfileByType returns a copy of profiles indexed by lowercase type (last wins if caller duped).
func AgentProfileByType(profiles []AgentProfile) map[string]AgentProfile {
	m := make(map[string]AgentProfile, len(profiles))
	for _, p := range profiles {
		m[strings.ToLower(strings.TrimSpace(p.Type))] = p
	}
	return m
}
