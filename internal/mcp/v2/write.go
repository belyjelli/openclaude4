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

// AppendServer merges a server into path (YAML or JSON). Creates file if missing.
func AppendServer(path string, srv Server) error {
	name := strings.TrimSpace(srv.Name)
	if name == "" {
		return errors.New("mcp server name is empty")
	}
	if len(srv.Command) == 0 {
		return errors.New("mcp server command is empty")
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = ".yaml"
	}

	var doc Doc
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(raw) > 0 {
		switch ext {
		case ".json":
			if err := json.Unmarshal(raw, &doc); err != nil {
				return fmt.Errorf("parse json: %w", err)
			}
		default:
			if err := yaml.Unmarshal(raw, &doc); err != nil {
				return fmt.Errorf("parse yaml: %w", err)
			}
		}
	}
	if strings.TrimSpace(doc.Version) == "" {
		doc.Version = "2"
	}
	if err := validateDoc(&doc); err != nil {
		return err
	}
	for _, row := range doc.Servers {
		if strings.TrimSpace(row.Name) == name {
			return fmt.Errorf("config already contains MCP server %q", name)
		}
	}
	row := serverRow{
		Name:      name,
		Transport: strings.TrimSpace(srv.Transport),
		Command:   append([]string(nil), srv.Command...),
		Env:       srv.Env,
		Approval:  srv.Approval,
		Policies:  srv.Policies,
	}
	if row.Transport == "" {
		row.Transport = "stdio"
	}
	if row.Env == nil {
		row.Env = map[string]string{}
	}
	doc.Servers = append(doc.Servers, row)

	var out []byte
	switch ext {
	case ".json":
		out, err = json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
	default:
		out, err = yaml.Marshal(doc)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// OverwriteFile replaces the entire MCP v2 file with the given servers (version 2).
func OverwriteFile(path string, servers []Server) error {
	doc := Doc{Version: "2"}
	for _, srv := range servers {
		name := strings.TrimSpace(srv.Name)
		if name == "" || len(srv.Command) == 0 {
			continue
		}
		t := strings.TrimSpace(srv.Transport)
		if t == "" {
			t = "stdio"
		}
		doc.Servers = append(doc.Servers, serverRow{
			Name:      name,
			Transport: t,
			Command:   append([]string(nil), srv.Command...),
			Env:       srv.Env,
			Approval:  srv.Approval,
			Policies:  srv.Policies,
		})
	}
	if err := validateDoc(&doc); err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = ".yaml"
	}
	var out []byte
	var err error
	switch ext {
	case ".json":
		out, err = json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
	default:
		out, err = yaml.Marshal(doc)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
