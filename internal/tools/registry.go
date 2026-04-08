package tools

import "github.com/gitlawb/openclaude4/internal/skills"

// NewDefaultRegistry registers built-in tools plus skill tools and Go outline.
// skillCatalog may be nil (treated as empty).
func NewDefaultRegistry(skillCatalog *skills.Catalog) *Registry {
	if skillCatalog == nil {
		skillCatalog = skills.EmptyCatalog()
	}
	r := NewRegistry()
	r.Register(FileRead{})
	r.Register(FileWrite{})
	r.Register(FileEdit{})
	r.Register(Bash{})
	r.Register(Grep{})
	r.Register(Glob{})
	r.Register(WebSearch{})
	r.Register(WebFetch{})
	registerSpiderIfAvailable(r)
	r.Register(SkillsList{Cat: skillCatalog})
	r.Register(SkillsRead{Cat: skillCatalog})
	r.Register(GoOutline{})
	return r
}
