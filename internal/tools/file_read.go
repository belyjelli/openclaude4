package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// FileRead reads a text file within the workspace (size-capped).
type FileRead struct{}

func (FileRead) Name() string      { return "FileRead" }
func (FileRead) IsDangerous() bool { return false }
func (FileRead) Description() string {
	return "Read the full text of a file under the workspace (UTF-8, capped at 512KiB)."
}

func (FileRead) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path relative to workspace or absolute within workspace",
			},
		},
		"required": []string{"file_path"},
	}
}

func (FileRead) Execute(ctx context.Context, args map[string]any) (string, error) {
	_ = ctx
	path, _ := args["file_path"].(string)
	if path == "" {
		return "", fmt.Errorf("file_path is required")
	}
	return ReadWorkspaceText(ctx, path)
}

// ReadWorkspaceText reads a UTF-8 text file under the workspace with the same rules and size cap as [FileRead].
func ReadWorkspaceText(ctx context.Context, relOrAbs string) (string, error) {
	if strings.TrimSpace(relOrAbs) == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := resolveUnderWorkdir(ctx, relOrAbs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", relOrAbs)
	}
	if info.Size() > maxReadFileBytes {
		return "", fmt.Errorf("file too large (%d bytes; max %d)", info.Size(), maxReadFileBytes)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
