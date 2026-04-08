// Package skills loads prompt-style skills from directories (SKILL.md per skill).
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Entry is one loaded skill.
type Entry struct {
	Name        string
	Description string
	Body        string
	Dir         string // absolute path to skill folder
}

// Catalog is a read-only snapshot of loaded skills.
type Catalog struct {
	byName map[string]Entry
	order  []string
}

// Load scans each root directory for subfolders containing SKILL.md (case-insensitive).
// Also loads root/SKILL.md as a skill named after the parent directory of the file.
func Load(roots []string) (*Catalog, error) {
	byName := make(map[string]Entry)
	seen := make(map[string]struct{})
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("skills dir %q: %w", root, err)
		}
		st, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("skills dir %q: %w", abs, err)
		}
		if !st.IsDir() {
			continue
		}
		if err := scanSkillRoot(abs, byName, seen); err != nil {
			return nil, err
		}
	}
	order := make([]string, 0, len(byName))
	for n := range byName {
		order = append(order, n)
	}
	sort.Strings(order)
	return &Catalog{byName: byName, order: order}, nil
}

func scanSkillRoot(root string, byName map[string]Entry, seen map[string]struct{}) error {
	// Optional: <root>/SKILL.md → skill name = base(root)
	if p, ok := findSkillFile(root); ok {
		name := filepath.Base(root)
		if err := addSkillFile(root, p, name, byName, seen); err != nil {
			return err
		}
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(root, e.Name())
		p, ok := findSkillFile(sub)
		if !ok {
			continue
		}
		if err := addSkillFile(sub, p, e.Name(), byName, seen); err != nil {
			return err
		}
	}
	return nil
}

func findSkillFile(dir string) (path string, ok bool) {
	for _, n := range []string{"SKILL.md", "skill.md", "Skill.md"} {
		p := filepath.Join(dir, n)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

func addSkillFile(skillDir, mdPath, defaultName string, byName map[string]Entry, seen map[string]struct{}) error {
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}
	name, desc, body, err := parseSkillMarkdown(raw, defaultName)
	if err != nil {
		return fmt.Errorf("%s: %w", mdPath, err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}
	if _, dup := seen[name]; dup {
		return fmt.Errorf("duplicate skill name %q (from %s)", name, mdPath)
	}
	seen[name] = struct{}{}
	byName[name] = Entry{
		Name:        name,
		Description: desc,
		Body:        body,
		Dir:         skillDir,
	}
	return nil
}

type fmYAML struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseSkillMarkdown(raw []byte, defaultName string) (name, description, body string, err error) {
	s := strings.TrimPrefix(string(raw), "\ufeff")
	fm, rest, ok := splitYAMLFrontmatter(s)
	if !ok {
		body = strings.TrimSpace(s)
		return defaultName, "", body, nil
	}
	var meta fmYAML
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return "", "", "", fmt.Errorf("frontmatter: %w", err)
	}
	n := strings.TrimSpace(meta.Name)
	if n == "" {
		n = defaultName
	}
	return n, strings.TrimSpace(meta.Description), strings.TrimSpace(rest), nil
}

func splitYAMLFrontmatter(s string) (frontmatter, body string, ok bool) {
	s = strings.TrimLeft(s, " \t\r\n")
	if !strings.HasPrefix(s, "---") {
		return "", "", false
	}
	s = strings.TrimPrefix(s, "---")
	s = strings.TrimLeft(s, "\r\n")
	// End of frontmatter: newline + --- + boundary
	idx := strings.Index(s, "\n---")
	if idx < 0 {
		return "", "", false
	}
	fm := strings.TrimSpace(s[:idx])
	after := s[idx+len("\n---"):]
	after = strings.TrimLeft(after, "\r\n")
	return fm, after, true
}

// EmptyCatalog returns an empty catalog (no skills).
func EmptyCatalog() *Catalog {
	return &Catalog{byName: make(map[string]Entry)}
}

// Len returns the number of skills.
func (c *Catalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byName)
}

// Names returns sorted skill names.
func (c *Catalog) Names() []string {
	if c == nil {
		return nil
	}
	out := make([]string, len(c.order))
	copy(out, c.order)
	return out
}

// Get returns a skill by name (exact match).
func (c *Catalog) Get(name string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	e, ok := c.byName[name]
	return e, ok
}

// GetFold returns a skill by case-insensitive name match.
func (c *Catalog) GetFold(name string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Entry{}, false
	}
	if e, ok := c.Get(name); ok {
		return e, true
	}
	lower := strings.ToLower(name)
	for _, n := range c.order {
		if strings.ToLower(n) == lower {
			return c.byName[n], true
		}
	}
	return Entry{}, false
}

// List returns entries in stable name order.
func (c *Catalog) List() []Entry {
	if c == nil {
		return nil
	}
	out := make([]Entry, 0, len(c.order))
	for _, n := range c.order {
		out = append(out, c.byName[n])
	}
	return out
}
