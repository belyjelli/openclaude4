package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileWrite overwrites or creates a file under the workspace.
type FileWrite struct{}

func (FileWrite) Name() string      { return "FileWrite" }
func (FileWrite) IsDangerous() bool { return true }
func (FileWrite) Description() string {
	return "Write UTF-8 text to a file (creates parent directories). Overwrites existing files."
}

func (FileWrite) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path relative to workspace or absolute within workspace",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full new file contents",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (FileWrite) Execute(ctx context.Context, args map[string]any) (string, error) {
	_ = ctx
	path, _ := args["file_path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return "", fmt.Errorf("file_path is required")
	}
	abs, err := resolveUnderWorkdir(ctx, path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}
