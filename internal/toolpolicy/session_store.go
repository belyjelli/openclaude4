package toolpolicy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// SessionStore persists extra allow rules beside the session JSON transcript.
type SessionStore struct {
	mu   sync.Mutex
	path string
}

// SessionPermissionsPath returns the per-session permissions file path, or empty if store is nil/incomplete.
func SessionPermissionsPath(sessionDir, sessionID string) string {
	sessionDir = strings.TrimSpace(sessionDir)
	sessionID = strings.TrimSpace(sessionID)
	if sessionDir == "" || sessionID == "" {
		return ""
	}
	safe := sanitizeID(sessionID)
	if safe == "" {
		return ""
	}
	return filepath.Join(sessionDir, safe+"_permissions.local.yaml")
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	var b strings.Builder
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

type localFile struct {
	Allow []string `yaml:"allow"`
}

// NewSessionStore returns a store bound to path (may be empty to disable persistence).
func NewSessionStore(path string) *SessionStore {
	return &SessionStore{path: strings.TrimSpace(path)}
}

// Load reads allow rules from disk (best effort).
func (s *SessionStore) Load() ([]string, error) {
	if s == nil || s.path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var f localFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(f.Allow))
	for _, r := range f.Allow {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out, nil
}

// AppendAllow merges new rules, dedupes, and writes the file (atomic rename).
func (s *SessionStore) AppendAllow(rules []string) error {
	if s == nil || s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing localFile
	if data, err := os.ReadFile(s.path); err == nil && len(data) > 0 {
		_ = yaml.Unmarshal(data, &existing)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	seen := map[string]struct{}{}
	for _, r := range existing.Allow {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		seen[strings.ToLower(r)] = struct{}{}
	}
	for _, r := range rules {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		k := strings.ToLower(r)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		existing.Allow = append(existing.Allow, r)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(&existing)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".perm-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
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
	return os.Rename(tmpPath, s.path)
}
