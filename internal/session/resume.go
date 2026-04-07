package session

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveResumeID picks which session to open for --resume:
// 1) last id written by [Store.Save] (last_session_id file)
// 2) otherwise the most recently modified *.json session file in dir.
func ResolveResumeID(dir string) (string, error) {
	if dir == "" {
		return "", os.ErrInvalid
	}
	if id, err := ReadLastSessionID(dir); err == nil && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id), nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", err
	}
	var bestPath string
	var bestMod os.FileInfo
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		full := filepath.Join(dir, name)
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			continue
		}
		if bestMod == nil || fi.ModTime().After(bestMod.ModTime()) {
			bestMod = fi
			bestPath = full
		}
	}
	if bestPath == "" {
		return "", os.ErrNotExist
	}
	base := filepath.Base(bestPath)
	return strings.TrimSuffix(base, ".json"), nil
}
