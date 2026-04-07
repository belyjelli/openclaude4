package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is a row for session listing.
type Entry struct {
	Name    string
	NMsgs   int
	Updated time.Time
	CWD     string
	Path    string
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
				Name: base + " (unreadable)",
				Path: full,
			})
			continue
		}
		id := data.ID
		if id == "" {
			id = base
		}
		out = append(out, Entry{
			Name:    id,
			NMsgs:   len(data.Messages),
			Updated: data.UpdatedAt,
			CWD:     data.WorkDir,
			Path:    full,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].Updated, out[j].Updated
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}
