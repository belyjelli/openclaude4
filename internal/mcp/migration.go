package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	mcpv2 "github.com/gitlawb/openclaude4/internal/mcp/v2"
	yaml "go.yaml.in/yaml/v3"
)

// MigrateLegacyToUserV2 writes ~/.openclaude/mcp.yaml from legacy [config.MCPServer] entries.
// If the target file exists and contains servers, returns [ErrMigrateNeedForce] unless force is true.
func MigrateLegacyToUserV2(home string, servers []config.MCPServer, force bool) (outPath string, err error) {
	if len(servers) == 0 {
		return "", errors.New("no mcp.servers entries in openclaude config to migrate")
	}
	if strings.TrimSpace(home) == "" {
		return "", errors.New("HOME is not set; cannot write ~/.openclaude/mcp.yaml")
	}
	dir := filepath.Join(home, ".openclaude")
	outPath = filepath.Join(dir, "mcp.yaml")
	if !force {
		if doc, err := mcpv2.LoadFile(outPath); err == nil && len(doc.Servers) > 0 {
			return outPath, ErrMigrateNeedForce
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var rows []mcpv2.Server
	for _, s := range servers {
		rows = append(rows, mcpv2.Server{
			Name:      s.Name,
			Transport: "stdio",
			Command:   append([]string(nil), s.Command...),
			Env:       s.Env,
			Approval:  config.NormalizeMCPApproval(s.Approval),
		})
	}
	if err := mcpv2.OverwriteFile(outPath, rows); err != nil {
		return outPath, err
	}
	return outPath, nil
}

// ErrMigrateNeedForce is returned when ~/.openclaude/mcp.yaml already has servers.
var ErrMigrateNeedForce = errors.New("~/.openclaude/mcp.yaml already contains MCP servers (use --force)")

// WriteMCPV1Backup writes legacy mcp.servers to a standalone YAML file (for user records).
func WriteMCPV1Backup(path string, servers []config.MCPServer) error {
	root := map[string]any{
		"mcp": map[string]any{
			"servers": legacyServersToYAML(servers),
		},
	}
	raw, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func legacyServersToYAML(servers []config.MCPServer) []any {
	var rows []any
	for _, s := range servers {
		row := map[string]any{
			"name":     s.Name,
			"command":  append([]string(nil), s.Command...),
			"approval": config.NormalizeMCPApproval(s.Approval),
		}
		if len(s.Env) > 0 {
			row["env"] = s.Env
		}
		rows = append(rows, row)
	}
	return rows
}
