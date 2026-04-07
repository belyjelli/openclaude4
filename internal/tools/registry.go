package tools

// NewDefaultRegistry registers Phase 1 tools.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(FileRead{})
	r.Register(FileWrite{})
	r.Register(FileEdit{})
	r.Register(Bash{})
	r.Register(Grep{})
	r.Register(Glob{})
	r.Register(WebSearch{})
	return r
}
