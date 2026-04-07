package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	grepMaxBytes   = 64 << 10
	grepMaxFiles   = 400
	grepMaxMatches = 200
)

// Grep searches file contents with a regular expression under a path.
type Grep struct{}

func (Grep) Name() string        { return "Grep" }
func (Grep) IsDangerous() bool   { return false }
func (Grep) Description() string { return "Search UTF-8 text files under a path using Go regexp syntax. Skips large/binary-looking files." }

func (Grep) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regular expression (Go RE2 syntax)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory relative to workspace (default \".\")",
			},
			"include_glob": map[string]any{
				"type":        "string",
				"description": "Optional filepath.Glob pattern relative to search path (e.g. \"*.go\")",
			},
		},
		"required": []string{"pattern"},
	}
}

func (Grep) Execute(ctx context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("regexp: %w", err)
	}

	relPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		relPath = p
	}
	include, _ := args["include_glob"].(string)

	root, err := resolveUnderWorkdir(ctx, relPath)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("path must be a directory")
	}

	var b strings.Builder
	matches := 0
	files := 0

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
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
		if files >= grepMaxFiles {
			return fs.SkipAll
		}
		if include != "" {
			ok, err := filepath.Match(include, filepath.Base(path))
			if err != nil || !ok {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() == 0 || info.Size() > 256*1024 {
			return nil
		}
		files++
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !looksText(data) {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		rel, _ := filepath.Rel(root, path)
		for i, line := range lines {
			if re.MatchString(line) {
				matches++
				fmt.Fprintf(&b, "%s:%d:%s\n", rel, i+1, line)
				if b.Len() > grepMaxBytes || matches >= grepMaxMatches {
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(no matches)", nil
	}
	return out, nil
}

func looksText(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for i := 0; i < len(b) && i < 8000; i++ {
		if b[i] == 0 {
			return false
		}
	}
	return true
}
