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

// Entry is one loaded skill (v3-shaped frontmatter where applicable).
type Entry struct {
	Name        string
	Description string
	Body        string
	Dir         string // absolute path to skill folder

	// Extended (v3-style); zero values mean unset / default.
	ArgumentNames          []string
	AllowedTools           []string
	Context                string // "fork", "inline", or ""
	Agent                  string
	Model                  string
	Effort                 string
	Shell                  string // "bash", "powershell", or ""
	DisableModelInvocation bool
	WhenToUse              string
	Version                string
	Paths                  []string
	Hooks                  map[string]any // v3-shaped hooks YAML subtree; nil if absent
	SourceLocal            bool           // true for on-disk skills (prompt shell allowed)
	MaxForkIterations      int            // optional cap for forked sub-agent; 0 = default
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
	ent, err := ParseSkillMarkdown(raw, defaultName, skillDir)
	if err != nil {
		return fmt.Errorf("%s: %w", mdPath, err)
	}
	name := strings.TrimSpace(ent.Name)
	if name == "" {
		name = defaultName
	}
	ent.Name = name
	if _, dup := seen[name]; dup {
		return fmt.Errorf("duplicate skill name %q (from %s)", name, mdPath)
	}
	seen[name] = struct{}{}
	byName[name] = ent
	return nil
}

type fmYAML struct {
	Name                   string         `yaml:"name"`
	Description            string         `yaml:"description"`
	Arguments              any            `yaml:"arguments"`
	AllowedTools           []string       `yaml:"allowed_tools"`
	Context                string         `yaml:"context"`
	Agent                  string         `yaml:"agent"`
	Model                  string         `yaml:"model"`
	Effort                 string         `yaml:"effort"`
	Shell                  string         `yaml:"shell"`
	DisableModelInvocation bool           `yaml:"disable_model_invocation"`
	WhenToUse              string         `yaml:"when_to_use"`
	Version                string         `yaml:"version"`
	Paths                  []string       `yaml:"paths"`
	MaxForkIterations      int            `yaml:"max_fork_iterations"`
	Hooks                  map[string]any `yaml:"hooks"`
}

// ParseSkillMarkdown parses SKILL.md into an Entry (including extended frontmatter).
func ParseSkillMarkdown(raw []byte, defaultName, skillDir string) (Entry, error) {
	s := strings.TrimPrefix(string(raw), "\ufeff")
	fm, rest, ok := splitYAMLFrontmatter(s)
	if !ok {
		return Entry{
			Name:        defaultName,
			Description: "",
			Body:        strings.TrimSpace(s),
			Dir:         skillDir,
			SourceLocal: true,
		}, nil
	}
	var meta fmYAML
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Entry{}, fmt.Errorf("frontmatter: %w", err)
	}
	n := strings.TrimSpace(meta.Name)
	if n == "" {
		n = defaultName
	}
	argNames := parseArgumentsField(meta.Arguments)
	hooks := meta.Hooks
	if len(hooks) == 0 {
		hooks = nil
	}
	ctx := strings.ToLower(strings.TrimSpace(meta.Context))
	if ctx != "fork" && ctx != "inline" {
		ctx = ""
	}
	return Entry{
		Name:                   n,
		Description:            strings.TrimSpace(meta.Description),
		Body:                   strings.TrimSpace(rest),
		Dir:                    skillDir,
		ArgumentNames:          argNames,
		AllowedTools:           append([]string(nil), meta.AllowedTools...),
		Context:                ctx,
		Agent:                  strings.TrimSpace(meta.Agent),
		Model:                  strings.TrimSpace(meta.Model),
		Effort:                 strings.TrimSpace(meta.Effort),
		Shell:                  strings.ToLower(strings.TrimSpace(meta.Shell)),
		DisableModelInvocation: meta.DisableModelInvocation,
		WhenToUse:              strings.TrimSpace(meta.WhenToUse),
		Version:                strings.TrimSpace(meta.Version),
		Paths:                  append([]string(nil), meta.Paths...),
		Hooks:                  hooks,
		SourceLocal:            true,
		MaxForkIterations:      meta.MaxForkIterations,
	}, nil
}

func parseArgumentsField(v any) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case string:
		return parseArgumentNamesFromString(x)
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s, _ := e.(string)
			s = strings.TrimSpace(s)
			if s != "" && !isNumericOnlyName(s) {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(x))
		for _, s := range x {
			s = strings.TrimSpace(s)
			if s != "" && !isNumericOnlyName(s) {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func parseArgumentNamesFromString(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		if w != "" && !isNumericOnlyName(w) {
			out = append(out, w)
		}
	}
	return out
}

func isNumericOnlyName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
