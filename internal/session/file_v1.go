package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdk "github.com/sashabaranov/go-openai"
)

const fileFormatVersion = 1

// FileV1 is the on-disk JSON shape for a saved session.
type FileV1 struct {
	Version   int                            `json:"version"`
	ID        string                         `json:"id"`
	UpdatedAt time.Time                      `json:"updated_at"`
	WorkDir   string                         `json:"work_dir,omitempty"`
	Messages  []sdk.ChatCompletionMessage    `json:"messages"`
}

// ErrUnsupportedVersion is returned when the file version is not 1.
var ErrUnsupportedVersion = errors.New("session: unsupported file version")

// ReadFileV1 parses a session JSON file. Partial or invalid JSON returns a non-nil error;
// callers should treat the file as unusable and start a fresh transcript.
func ReadFileV1(path string) (FileV1, error) {
	var z FileV1
	raw, err := os.ReadFile(path)
	if err != nil {
		return z, err
	}
	if len(raw) == 0 {
		return z, errors.New("session: empty file")
	}
	if err := json.Unmarshal(raw, &z); err != nil {
		return z, fmt.Errorf("session: decode: %w", err)
	}
	if z.Version != 0 && z.Version != fileFormatVersion {
		return z, fmt.Errorf("%w: %d", ErrUnsupportedVersion, z.Version)
	}
	if z.Version == 0 {
		z.Version = fileFormatVersion
	}
	return z, nil
}

// WriteFileV1 atomically writes the session file (temp + rename).
func WriteFileV1(path string, snap FileV1) error {
	snap.Version = fileFormatVersion
	if snap.UpdatedAt.IsZero() {
		snap.UpdatedAt = time.Now().UTC()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".session-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	_, werr := f.Write(payload)
	cerr := f.Close()
	if werr != nil {
		_ = os.Remove(tmpPath)
		return werr
	}
	if cerr != nil {
		_ = os.Remove(tmpPath)
		return cerr
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
