package output

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// PostProcess merges streams (caller already merged), applies size cap, persists overflow.
func PostProcess(workspace, command string, combined []byte, inlineMax int) (text string, persistedPath string, err error) {
	if inlineMax <= 0 {
		inlineMax = 30 * 1024
	}
	// Binary sniff: non-text → short message
	if len(combined) > 0 && !utf8.Valid(combined) {
		p := ""
		if workspace != "" {
			var werr error
			p, werr = persistOutput(workspace, combined)
			if werr != nil {
				return "", "", werr
			}
		}
		return fmt.Sprintf("(binary output, %d bytes, saved to %s)", len(combined), p), p, nil
	}
	s := strings.TrimRight(string(combined), "\n\r")
	if len(combined) <= inlineMax {
		return s, "", nil
	}
	p, err := persistOutput(workspace, combined)
	if err != nil {
		return s, "", err
	}
	head := s
	if len(head) > 2048 {
		head = head[:2048] + "\n…"
	}
	return fmt.Sprintf("%s\n\n(output truncated: %d bytes total, full output saved to %s)", head, len(combined), p), p, nil
}

func persistOutput(workspace string, data []byte) (string, error) {
	dir := filepath.Join(workspace, ".openclaude", "tmp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	name := "bash-out-" + hex.EncodeToString(sum[:10]) + ".txt"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return "", err
	}
	return p, nil
}

// SemanticSuccess treats grep "no match" exit 1 as non-error for the model-facing message.
func SemanticSuccess(command string, exit int) bool {
	if exit == 0 {
		return true
	}
	if exit != 1 {
		return false
	}
	cmd := strings.TrimSpace(strings.ToLower(command))
	if strings.HasPrefix(cmd, "grep ") || strings.HasPrefix(cmd, "git grep ") {
		return true
	}
	return false
}
