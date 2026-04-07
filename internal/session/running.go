package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RunningMeta is written to the session dir when a chat process starts (for ps-style listing).
type RunningMeta struct {
	PID       int    `json:"pid"`
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Started   string `json:"started"` // RFC3339
	TUI       bool   `json:"tui"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
}

// RunningEntry is one row from [ListRunning].
type RunningEntry struct {
	Meta  RunningMeta
	Path  string
	Alive bool
}

// RegisterRunning writes running/<pid>.json and returns a cleanup that removes it.
func RegisterRunning(sessionDir string, meta RunningMeta) (func(), error) {
	if sessionDir == "" {
		return nil, errors.New("session dir empty")
	}
	if meta.PID <= 0 {
		meta.PID = os.Getpid()
	}
	if strings.TrimSpace(meta.Started) == "" {
		meta.Started = time.Now().UTC().Format(time.RFC3339)
	}
	runDir := filepath.Join(sessionDir, "running")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir running registry: %w", err)
	}
	p := filepath.Join(runDir, fmt.Sprintf("%d.json", meta.PID))
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return nil, fmt.Errorf("write running record: %w", err)
	}
	return func() { _ = os.Remove(p) }, nil
}

// ListRunning reads running/*.json under sessionDir and checks whether each PID is still alive.
func ListRunning(sessionDir string) ([]RunningEntry, error) {
	if sessionDir == "" {
		return nil, errors.New("session dir empty")
	}
	runDir := filepath.Join(sessionDir, "running")
	ents, err := os.ReadDir(runDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []RunningEntry
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		full := filepath.Join(runDir, name)
		raw, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		var meta RunningMeta
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}
		out = append(out, RunningEntry{
			Meta:  meta,
			Path:  full,
			Alive: pidAlive(meta.PID),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, out[i].Meta.Started)
		tj, _ := time.Parse(time.RFC3339, out[j].Meta.Started)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].Meta.PID < out[j].Meta.PID
	})
	return out, nil
}
