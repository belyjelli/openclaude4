package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
	"github.com/spf13/viper"
)

// WritableConfigPath picks where to persist changes (e.g. [AppendMCPServerToConfigFile]).
// Order: viper's loaded config file; else first existing openclaude.{yaml,yml,json} in search dirs;
// else ~/.config/openclaude/openclaude.yaml (directory created); else ./openclaude.yaml in cwd.
func WritableConfigPath() (string, error) {
	if p := viper.ConfigFileUsed(); strings.TrimSpace(p) != "" {
		return filepath.Abs(p)
	}
	home, _ := os.UserHomeDir()
	for _, base := range configSearchDirs(home) {
		for _, ext := range []string{"yaml", "yml", "json"} {
			candidate := filepath.Join(base, "openclaude."+ext)
			st, err := os.Stat(candidate)
			if err != nil || st.IsDir() {
				continue
			}
			return filepath.Abs(candidate)
		}
	}
	if home != "" {
		dir := filepath.Join(home, ".config", "openclaude")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		return filepath.Join(dir, "openclaude.yaml"), nil
	}
	return filepath.Abs("openclaude.yaml")
}

// AppendMCPServerToConfigFile reads path (YAML or JSON), appends srv under mcp.servers, and writes back.
// Duplicate server names in the file return [ErrMCPNameExists]. Comments in YAML are not preserved.
func AppendMCPServerToConfigFile(path string, srv MCPServer) error {
	name := strings.TrimSpace(srv.Name)
	if name == "" {
		return errors.New("mcp server name is empty")
	}
	if len(srv.Command) == 0 {
		return errors.New("mcp server command is empty")
	}
	ap := NormalizeMCPApproval(srv.Approval)

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = ".yaml"
	}

	var root map[string]any
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		raw = nil
	}
	if len(raw) > 0 {
		switch ext {
		case ".json":
			if err := json.Unmarshal(raw, &root); err != nil {
				return fmt.Errorf("parse json: %w", err)
			}
		default:
			if err := yaml.Unmarshal(raw, &root); err != nil {
				return fmt.Errorf("parse yaml: %w", err)
			}
		}
	}
	if root == nil {
		root = map[string]any{}
	}

	mcpBlock, ok := root["mcp"].(map[string]any)
	if !ok || mcpBlock == nil {
		mcpBlock = map[string]any{}
		root["mcp"] = mcpBlock
	}
	serversAny, ok := mcpBlock["servers"].([]any)
	if !ok || serversAny == nil {
		serversAny = []any{}
	}
	for _, row := range serversAny {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(m["name"])) == name {
			return ErrMCPNameExists
		}
	}

	entry := map[string]any{
		"name":     name,
		"command":  append([]string(nil), srv.Command...),
		"approval": ap,
	}
	if len(srv.Env) > 0 {
		env := make(map[string]any, len(srv.Env))
		for k, v := range srv.Env {
			env[k] = v
		}
		entry["env"] = env
	}
	serversAny = append(serversAny, entry)
	mcpBlock["servers"] = serversAny

	var out []byte
	switch ext {
	case ".json":
		out, err = json.MarshalIndent(root, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
	default:
		out, err = yaml.Marshal(root)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// ErrMCPNameExists is returned when the config file already lists an MCP server with the same name.
var ErrMCPNameExists = errors.New("config already contains an MCP server with this name")

// NormalizeMCPApproval returns ask, always, or never for MCP server approval settings.
func NormalizeMCPApproval(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "always", "never":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "ask"
	}
}
