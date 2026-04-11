package tools

import (
	"context"
	"encoding/json"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Tool is one model-callable capability.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
	IsDangerous() bool
}

// Registry holds registered tools by function name.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool; panics if name collides (startup error).
func (r *Registry) Register(t Tool) {
	name := t.Name()
	if _, ok := r.tools[name]; ok {
		panic("duplicate tool: " + name)
	}
	r.tools[name] = t
}

// Get returns a tool by OpenAI function name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all tools in arbitrary order.
func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// CloneRegistryOmit returns a new registry with every tool from src except those named omit.
func CloneRegistryOmit(src *Registry, omit string) *Registry {
	out := NewRegistry()
	if src == nil {
		return out
	}
	for _, t := range src.List() {
		if t.Name() == omit {
			continue
		}
		out.Register(t)
	}
	return out
}

// CloneRegistry returns a shallow copy of all tools in src.
func CloneRegistry(src *Registry) *Registry {
	out := NewRegistry()
	if src == nil {
		return out
	}
	for _, t := range src.List() {
		out.Register(t)
	}
	return out
}

// CloneRegistryAllow returns a registry containing only tools whose names appear in allow.
// If allow is empty, returns CloneRegistry(src) (no restriction).
func CloneRegistryAllow(src *Registry, allow []string) *Registry {
	if src == nil {
		return NewRegistry()
	}
	if len(allow) == 0 {
		return CloneRegistry(src)
	}
	want := make(map[string]struct{}, len(allow))
	for _, a := range allow {
		a = strings.TrimSpace(a)
		if a != "" {
			want[a] = struct{}{}
		}
	}
	out := NewRegistry()
	for _, t := range src.List() {
		if _, ok := want[t.Name()]; ok {
			out.Register(t)
		}
	}
	return out
}

// OpenAITools converts registry tools to the SDK tool list.
func OpenAITools(reg *Registry) ([]sdk.Tool, error) {
	list := reg.List()
	out := make([]sdk.Tool, 0, len(list))
	for _, t := range list {
		raw, err := json.Marshal(t.Parameters())
		if err != nil {
			return nil, err
		}
		var params jsonschema.Definition
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		name := t.Name()
		desc := t.Description()
		out = append(out, sdk.Tool{
			Type: sdk.ToolTypeFunction,
			Function: &sdk.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters:  params,
			},
		})
	}
	return out, nil
}
