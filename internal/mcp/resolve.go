package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gitlawb/openclaude4/internal/config"
	mcpv2 "github.com/gitlawb/openclaude4/internal/mcp/v2"
	"github.com/spf13/viper"
)

// ResolveSource indicates where MCP server definitions came from.
type ResolveSource string

const (
	SourceV2     ResolveSource = "v2"
	SourceLegacy ResolveSource = "legacy"
)

var legacyWarnOnce sync.Once

// ResolveOpts configures MCP v2 cascade resolution.
type ResolveOpts struct {
	Cwd           string
	MCPConfigPath string // viper: mcp.config_path or flag
}

// EffectiveServers returns MCP servers for runtime: v2 cascade when present, otherwise legacy viper mcp.servers.
func EffectiveServers(opts ResolveOpts) ([]ServerConfig, ResolveSource) {
	path := strings.TrimSpace(opts.MCPConfigPath)
	if path == "" {
		path = strings.TrimSpace(viper.GetString("mcp.config_path"))
	}
	cascadeOpts := mcpv2.CascadeOpts{Cwd: opts.Cwd, DynamicPath: path}

	v2rows, err := mcpv2.LoadCascade(cascadeOpts)
	if err == nil && len(v2rows) > 0 {
		return serversFromV2(v2rows), SourceV2
	}
	// If any v2 file exists but yields zero servers, do not fall back (explicit empty project).
	if mcpv2.HasConfigFile(cascadeOpts) && err == nil {
		return nil, SourceV2
	}

	legacy := config.MCPServers()
	if len(legacy) == 0 {
		return nil, SourceLegacy
	}
	out := make([]ServerConfig, 0, len(legacy))
	for _, s := range legacy {
		out = append(out, ServerConfig{
			Name:      s.Name,
			Transport: "stdio",
			Command:   append([]string(nil), s.Command...),
			Env:       s.Env,
			Approval:  s.Approval,
		})
	}
	return out, SourceLegacy
}

// WarnLegacyMCPOnce prints a one-time stderr notice when legacy openclaude.yaml mcp.servers is in use.
func WarnLegacyMCPOnce(print func(string)) {
	if print == nil {
		print = func(string) {}
	}
	legacyWarnOnce.Do(func() {
		print("openclaude: MCP v1 config (mcp.servers in openclaude.yaml) is deprecated. Use MCP v2 files: ~/.openclaude/mcp.yaml or .mcp.v2.yaml (version: \"2\"). Run: openclaude mcp migrate\n")
	})
}

// ResolveFromEnvironment loads MCP servers using the current working directory and viper flags/env.
func ResolveFromEnvironment() ([]ServerConfig, ResolveSource, error) {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	wd, err = filepath.Abs(wd)
	if err != nil {
		return nil, SourceLegacy, err
	}
	s, src := EffectiveServers(ResolveOpts{Cwd: wd, MCPConfigPath: viper.GetString("mcp.config_path")})
	return s, src, nil
}

func serversFromV2(rows []mcpv2.Server) []ServerConfig {
	out := make([]ServerConfig, 0, len(rows))
	for _, r := range rows {
		out = append(out, ServerConfig{
			Name:       r.Name,
			Transport:  r.Transport,
			Command:    r.Command,
			Env:        r.Env,
			Approval:   r.Approval,
			Policies:   r.Policies,
			ConfigPath: r.ConfigPath,
		})
	}
	return out
}
