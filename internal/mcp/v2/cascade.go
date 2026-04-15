package v2

import (
	"os"
	"path/filepath"
	"strings"
)

// CascadeOpts configures optional dynamic config path and working directory.
type CascadeOpts struct {
	Cwd          string
	DynamicPath  string // --mcp-config / OPENCLAUDE_MCP_CONFIG
	ExplicitPath string // when set, only this file is used (testing)
}

// LoadCascade merges MCP v2 layers: user (~/.openclaude/mcp.yaml), project (.mcp.v2.yaml root→cwd), dynamic file.
func LoadCascade(opts CascadeOpts) ([]Server, error) {
	cwd := strings.TrimSpace(opts.Cwd)
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}
	cwd, _ = filepath.Abs(cwd)

	if p := strings.TrimSpace(opts.ExplicitPath); p != "" {
		doc, err := LoadFile(p)
		if err != nil {
			return nil, err
		}
		return DocToServers(p, doc), nil
	}

	var layers [][]Server

	home, _ := os.UserHomeDir()
	if home != "" {
		userPath := filepath.Join(home, ".openclaude", "mcp.yaml")
		if doc, err := LoadFile(userPath); err == nil && len(doc.Servers) > 0 {
			layers = append(layers, DocToServers(userPath, doc))
		}
	}

	chain := ancestorDirs(cwd)
	for _, dir := range chain {
		p := filepath.Join(dir, ".mcp.v2.yaml")
		doc, err := LoadFile(p)
		if err != nil {
			continue
		}
		if len(doc.Servers) == 0 {
			continue
		}
		layers = append(layers, DocToServers(p, doc))
	}

	if p := strings.TrimSpace(opts.DynamicPath); p != "" {
		if doc, err := LoadFile(p); err == nil {
			layers = append(layers, DocToServers(p, doc))
		}
	}

	if len(layers) == 0 {
		return nil, nil
	}
	return MergeServerLayers(layers...), nil
}

func ancestorDirs(dir string) []string {
	dir = filepath.Clean(dir)
	var parts []string
	for d := dir; d != "" && d != string(filepath.Separator) && d != "."; {
		parts = append(parts, d)
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}

// HasConfigFile reports whether any MCP v2 config file exists on disk for cwd.
func HasConfigFile(opts CascadeOpts) bool {
	srv, err := LoadCascade(opts)
	if err != nil {
		return false
	}
	if len(srv) > 0 {
		return true
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		userPath := filepath.Join(home, ".openclaude", "mcp.yaml")
		if doc, err := LoadFile(userPath); err == nil && strings.TrimSpace(doc.Version) == "2" {
			return true
		}
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	for _, d := range ancestorDirs(cwd) {
		p := filepath.Join(d, ".mcp.v2.yaml")
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	if p := strings.TrimSpace(opts.DynamicPath); p != "" {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return len(srv) > 0
}
