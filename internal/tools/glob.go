package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const globMaxMatches = 500

// Glob lists files matching a glob pattern (supports **).
type Glob struct{}

func (Glob) Name() string        { return "Glob" }
func (Glob) IsDangerous() bool   { return false }
func (Glob) Description() string { return "List files under the workspace matching a glob (e.g. \"**/*.go\"). Uses doublestar semantics from the workspace root." }

func (Glob) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern from workspace root, e.g. \"**/*.md\"",
			},
		},
		"required": []string{"pattern"},
	}
}

func (Glob) Execute(ctx context.Context, args map[string]any) (string, error) {
	pat, _ := args["pattern"].(string)
	if pat == "" {
		return "", fmt.Errorf("pattern is required")
	}

	root := WorkDir(ctx)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = wd
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}

	var matches []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		ok, err := doublestar.Match(pat, rel)
		if err != nil || !ok {
			return nil
		}
		matches = append(matches, rel)
		if len(matches) >= globMaxMatches {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "(no matches)", nil
	}
	sort.Strings(matches)
	return strings.Join(matches, "\n"), nil
}
