package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Entry is one JSON line for Gate or Execute.
type Entry struct {
	TS            time.Time `json:"ts"`
	Phase         string    `json:"phase"` // gate|execute
	Reason        string    `json:"reason,omitempty"`
	SnapshotVer   string    `json:"snapshotVersion,omitempty"`
	Workspace     string    `json:"workspace,omitempty"`
	CWD           string    `json:"cwd,omitempty"`
	CommandHash   string    `json:"commandHash,omitempty"`
	ExitCode      int       `json:"exitCode,omitempty"`
	OutputBytes   int       `json:"outputBytes,omitempty"`
	PersistedPath string    `json:"persistedPath,omitempty"`
	Sandbox       string    `json:"sandbox,omitempty"`
	ToolCallID    string    `json:"toolCallId,omitempty"`
}

// Sink appends audit records.
type Sink struct {
	path string
	mu   sync.Mutex
}

// NewSink returns a file-backed sink or nil if path is empty.
func NewSink(path string) *Sink {
	if path == "" {
		return nil
	}
	return &Sink{path: path}
}

func (s *Sink) Log(e Entry) {
	if s == nil {
		return
	}
	e.TS = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}
