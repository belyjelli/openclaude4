package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

// Handle is a named on-disk transcript (JSON under Dir).
type Handle struct {
	Dir  string
	Name string
}

// NewHandle returns a handle for the given directory and session name (trimmed; must be non-empty).
func NewHandle(dir, name string) (*Handle, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("session: empty session name")
	}
	d := strings.TrimSpace(dir)
	if d == "" {
		return nil, errors.New("session: empty directory")
	}
	return &Handle{Dir: filepath.Clean(d), Name: name}, nil
}

// Path is the absolute path to the session JSON file.
func (h *Handle) Path() string {
	if h == nil {
		return ""
	}
	st := &Store{Dir: h.Dir, ID: h.Name}
	return st.SessionPath()
}

// SaveFrom writes the transcript and updates the last-session marker.
func (h *Handle) SaveFrom(msgs []sdk.ChatCompletionMessage, workDir string) error {
	if h == nil {
		return nil
	}
	st := &Store{Dir: h.Dir, ID: h.Name}
	return st.Save(RepairTranscript(msgs), workDir)
}

// LoadInto replaces *dest with repaired messages from disk, or clears dest if the file is missing.
func (h *Handle) LoadInto(dest *[]sdk.ChatCompletionMessage) error {
	if h == nil || dest == nil {
		return errors.New("session: nil handle or destination")
	}
	st := &Store{Dir: h.Dir, ID: h.Name}
	data, err := st.Load()
	if err != nil {
		if os.IsNotExist(err) {
			*dest = nil
			return nil
		}
		return err
	}
	*dest = RepairTranscript(data.Messages)
	return nil
}

// LatestName returns the session id recorded as "last used" (see [Store.Save]).
func LatestName(dir string) (string, error) {
	return ReadLastSessionID(dir)
}
