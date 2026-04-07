package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxReadFileBytes = 512 * 1024

func resolveUnderWorkdir(ctx context.Context, relOrAbs string) (string, error) {
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

	var p string
	if filepath.IsAbs(relOrAbs) {
		p = filepath.Clean(relOrAbs)
	} else {
		p = filepath.Clean(filepath.Join(root, relOrAbs))
	}

	rel, err := filepath.Rel(root, p)
	if err != nil {
		return "", fmt.Errorf("path %q: %w", relOrAbs, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace %q", relOrAbs, root)
	}
	return p, nil
}
