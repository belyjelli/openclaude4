package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a row for session listing.
type Entry struct {
	ID        string
	UpdatedAt string
	WorkDir   string
	Path      string
	Messages  int
}

// List scans dir for *.json session files (excluding malformed names).
func List(dir string) ([]Entry, error) {
	if dir == "" {
		return nil, os.ErrInvalid
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		if base == "" {
			continue
		}
		full := filepath.Join(dir, name)
		data, err := ReadFileV1(full)
		if err != nil {
			out = append(out, Entry{
				ID:   base + " (unreadable)",
				Path: full,
			})
			continue
		}
		id := data.ID
		if id == "" {
			id = base
		}
		updated := ""
		if !data.UpdatedAt.IsZero() {
			updated = data.UpdatedAt.UTC().Format(timeRFC3339Minute)
		}
		out = append(out, Entry{
			ID:        id,
			UpdatedAt: updated,
			WorkDir:   data.WorkDir,
			Path:      full,
			Messages:  len(data.Messages),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out, nil
}

const timeRFC3339Minute = "2006-01-02 15:04 UTC"
