package session

import (
	"os"
	"path/filepath"
)

// DefaultDir returns ~/.local/share/openclaude/sessions. It does not read OPENCLAUDE_SESSION_DIR;
// prefer [github.com/gitlawb/openclaude4/internal/config.SessionDir] for CLI behavior.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "openclaude", "sessions"), nil
}
