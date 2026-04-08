package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gitlawb/openclaude4/internal/skills"
)

// SkillsList lists discovered skills (name + description) as JSON.
type SkillsList struct {
	Cat *skills.Catalog
}

func (s SkillsList) Name() string { return "SkillsList" }

func (s SkillsList) IsDangerous() bool { return false }

func (s SkillsList) Description() string {
	return "List available user skills loaded from skill directories (SKILL.md). Returns JSON array of {name, description}."
}

func (s SkillsList) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (s SkillsList) Execute(_ context.Context, _ map[string]any) (string, error) {
	cat := s.Cat
	if cat == nil {
		cat = skills.EmptyCatalog()
	}
	type row struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	rows := make([]row, 0, cat.Len())
	for _, e := range cat.List() {
		rows = append(rows, row{Name: e.Name, Description: e.Description})
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SkillsRead returns the markdown body of one skill by exact name.
type SkillsRead struct {
	Cat *skills.Catalog
}

func (SkillsRead) Name() string { return "SkillsRead" }

func (SkillsRead) IsDangerous() bool { return false }

func (SkillsRead) Description() string {
	return "Load the full instructions for a named skill (from SkillsList). Use before applying a skill workflow."
}

func (SkillsRead) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Exact skill name from SkillsList",
			},
		},
		"required": []string{"name"},
	}
}

func (s SkillsRead) Execute(_ context.Context, args map[string]any) (string, error) {
	name := strings.TrimSpace(fmt.Sprint(args["name"]))
	if name == "" {
		return "", fmt.Errorf("skills_read: name is required")
	}
	cat := s.Cat
	if cat == nil {
		cat = skills.EmptyCatalog()
	}
	e, ok := cat.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown skill %q (use SkillsList)", name)
	}
	var b strings.Builder
	if e.Description != "" {
		b.WriteString("# ")
		b.WriteString(e.Name)
		b.WriteString("\n\n")
		b.WriteString(e.Description)
		b.WriteString("\n\n---\n\n")
	}
	b.WriteString(e.Body)
	return b.String(), nil
}
