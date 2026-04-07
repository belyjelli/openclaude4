package config

import (
	"strings"

	"github.com/spf13/viper"
)

// MCPServer describes one stdio MCP server subprocess (see docs/CONFIG.md).
type MCPServer struct {
	Name     string            `mapstructure:"name"`
	Command  []string          `mapstructure:"command"`
	Env      map[string]string `mapstructure:"env"`
	Approval string            `mapstructure:"approval"` // ask | always | never
}

// MCPServers returns configured MCP servers from `mcp.servers` in viper (YAML/JSON/env).
func MCPServers() []MCPServer {
	var wrap struct {
		Servers []MCPServer `mapstructure:"servers"`
	}
	if err := viper.UnmarshalKey("mcp", &wrap); err != nil {
		return nil
	}
	out := make([]MCPServer, 0, len(wrap.Servers))
	for _, s := range wrap.Servers {
		if strings.TrimSpace(s.Name) == "" || len(s.Command) == 0 {
			continue
		}
		out = append(out, s)
	}
	return out
}
