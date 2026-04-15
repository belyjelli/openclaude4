package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// Doc is the on-disk MCP v2 configuration document.
type Doc struct {
	Version string      `json:"version" yaml:"version"`
	Servers []serverRow `json:"servers" yaml:"servers"`
}

type serverRow struct {
	Name      string            `json:"name" yaml:"name"`
	Transport string            `json:"transport" yaml:"transport"`
	Command   []string          `json:"command" yaml:"command"`
	Env       map[string]string `json:"env" yaml:"env"`
	Approval  string            `json:"approval" yaml:"approval"`
	Policies  map[string]any    `json:"policies" yaml:"policies"`
}

// LoadFile reads a single MCP v2 YAML or JSON file.
func LoadFile(path string) (Doc, error) {
	var zero Doc
	raw, err := os.ReadFile(path)
	if err != nil {
		return zero, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var doc Doc
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &doc); err != nil {
			return zero, fmt.Errorf("parse json: %w", err)
		}
	default:
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return zero, fmt.Errorf("parse yaml: %w", err)
		}
	}
	if err := validateDoc(&doc); err != nil {
		return zero, err
	}
	return doc, nil
}

func validateDoc(doc *Doc) error {
	if doc == nil {
		return errors.New("nil doc")
	}
	v := strings.TrimSpace(doc.Version)
	if v != "" && v != "2" {
		return fmt.Errorf("unsupported mcp config version %q (expected 2)", v)
	}
	for _, s := range doc.Servers {
		if strings.TrimSpace(s.Name) == "" {
			return errors.New("mcp v2 server missing name")
		}
		if len(s.Command) == 0 {
			return fmt.Errorf("mcp v2 server %q: command is required", s.Name)
		}
	}
	return nil
}

// MergeServerLayers merges later layers over earlier for the same server name (last wins).
func MergeServerLayers(layers ...[]Server) []Server {
	merged := map[string]Server{}
	var order []string
	for _, layer := range layers {
		for _, sc := range layer {
			name := strings.TrimSpace(sc.Name)
			if name == "" {
				continue
			}
			if _, existed := merged[name]; existed {
				filtered := order[:0]
				for _, n := range order {
					if n != name {
						filtered = append(filtered, n)
					}
				}
				order = append(filtered, name)
			} else {
				order = append(order, name)
			}
			merged[name] = sc
		}
	}
	out := make([]Server, 0, len(order))
	for _, name := range order {
		out = append(out, merged[name])
	}
	return out
}

// DocToServers converts a loaded doc to a slice in file order with source path stamped.
func DocToServers(path string, doc Doc) []Server {
	var out []Server
	for _, row := range doc.Servers {
		name := strings.TrimSpace(row.Name)
		if name == "" || len(row.Command) == 0 {
			continue
		}
		t := strings.TrimSpace(row.Transport)
		if t == "" {
			t = "stdio"
		}
		cmd := append([]string(nil), row.Command...)
		for i := range cmd {
			cmd[i] = os.Expand(cmd[i], os.Getenv)
		}
		env := map[string]string{}
		for k, v := range row.Env {
			env[k] = os.Expand(v, os.Getenv)
		}
		out = append(out, Server{
			Name:       name,
			Transport:  t,
			Command:    cmd,
			Env:        env,
			Approval:   row.Approval,
			Policies:   row.Policies,
			ConfigPath: path,
		})
	}
	return out
}
