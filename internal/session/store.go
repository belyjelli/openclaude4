package session

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	sdk "github.com/sashabaranov/go-openai"
)

const lastSessionFileName = "last_session_id"

// Store writes session snapshots under Dir for a stable ID.
type Store struct {
	Dir string
	ID  string
}

// SessionPath returns the JSON path for the store ID.
func (s *Store) SessionPath() string {
	if s == nil || s.Dir == "" || s.ID == "" {
		return ""
	}
	safe := sanitizeSessionID(s.ID)
	if safe == "" {
		return ""
	}
	return filepath.Join(s.Dir, safe+".json")
}

// Load reads the session file if it exists.
func (s *Store) Load() (FileV1, error) {
	var z FileV1
	if s == nil {
		return z, os.ErrInvalid
	}
	p := s.SessionPath()
	if p == "" {
		return z, os.ErrNotExist
	}
	fi, err := os.Stat(p)
	if err != nil {
		return z, err
	}
	if fi.IsDir() {
		return z, os.ErrInvalid
	}
	data, err := ReadFileV1(p)
	if err != nil {
		return z, err
	}
	if data.ID == "" {
		data.ID = s.ID
	}
	return data, nil
}

// Save writes messages and updates the last-session marker.
func (s *Store) Save(msgs []sdk.ChatCompletionMessage, workDir string) error {
	if s == nil || s.ID == "" || s.Dir == "" {
		return nil
	}
	p := s.SessionPath()
	if p == "" {
		return nil
	}
	snap := FileV1{
		ID:        s.ID,
		UpdatedAt: time.Now().UTC(),
		WorkDir:   workDir,
		Messages:  msgs,
	}
	if err := WriteFileV1(p, snap); err != nil {
		return err
	}
	return writeLastSessionID(s.Dir, s.ID)
}

func writeLastSessionID(dir, id string) error {
	if dir == "" || id == "" {
		return nil
	}
	path := filepath.Join(dir, lastSessionFileName)
	tmp, err := os.CreateTemp(dir, ".last-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(strings.TrimSpace(id) + "\n"); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// ReadLastSessionID returns the id stored by [Store.Save] (or empty).
func ReadLastSessionID(dir string) (string, error) {
	if dir == "" {
		return "", os.ErrNotExist
	}
	raw, err := os.ReadFile(filepath.Join(dir, lastSessionFileName))
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(raw))
	if id == "" {
		return "", os.ErrNotExist
	}
	return id, nil
}

func sanitizeSessionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(id))
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
