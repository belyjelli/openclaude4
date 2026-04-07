package subtask

import (
	"context"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
)

type stubTool struct {
	name string
}

func (s stubTool) Name() string                       { return s.name }
func (stubTool) Description() string                   { return "" }
func (stubTool) Parameters() map[string]any           { return map[string]any{"type": "object"} }
func (stubTool) Execute(context.Context, map[string]any) (string, error) {
	return "", nil
}
func (stubTool) IsDangerous() bool { return false }

func TestCloneRegistryOmit(t *testing.T) {
	r := tools.NewRegistry()
	r.Register(tools.FileRead{})
	r.Register(stubTool{name: "Task"})

	child := cloneRegistryOmit(r, taskToolName)
	if _, ok := child.Get("Task"); ok {
		t.Fatal("child should not contain Task")
	}
	if _, ok := child.Get("FileRead"); !ok {
		t.Fatal("child should contain FileRead")
	}
}
