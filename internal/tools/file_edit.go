package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// FileEdit replaces one unique occurrence of old_string with new_string in a file.
type FileEdit struct{}

func (FileEdit) Name() string      { return "FileEdit" }
func (FileEdit) IsDangerous() bool { return true }
func (FileEdit) Description() string {
	return "Edit a file by replacing exactly one occurrence of old_string with new_string (plain text)."
}

func (FileEdit) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path relative to workspace or absolute within workspace",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "Snippet to replace (must appear exactly once)",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Replacement text (may be empty)",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

func (FileEdit) Execute(ctx context.Context, args map[string]any) (string, error) {
	_ = ctx
	path, _ := args["file_path"].(string)
	oldS, _ := args["old_string"].(string)
	newS, _ := args["new_string"].(string)
	if path == "" || oldS == "" {
		return "", fmt.Errorf("file_path and old_string are required")
	}
	abs, err := resolveUnderWorkdir(ctx, path)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	s := string(b)
	n := strings.Count(s, oldS)
	switch n {
	case 0:
		return "", fmt.Errorf("old_string not found in file")
	case 1:
	default:
		return "", fmt.Errorf("old_string matches %d times; need exactly one", n)
	}
	out := strings.Replace(s, oldS, newS, 1)
	if err := os.WriteFile(abs, []byte(out), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("updated %s (1 replacement)", path), nil
}
