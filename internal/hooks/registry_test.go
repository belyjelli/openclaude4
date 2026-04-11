package hooks

import (
	"context"
	"testing"
)

func TestRegisterAndDispatchUserPrompt(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	r := &Registry{
		sessions: make(map[string]map[SkillHookEvent][]matcher),
		roots:    make(map[string]string),
	}
	raw := map[string]any{
		"UserPromptSubmit": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{"type": "command", "command": "exit 0"},
				},
			},
		},
	}
	if err := r.RegisterFromSkill("sid", root, raw); err != nil {
		t.Fatal(err)
	}
	if err := r.Dispatch(context.Background(), "sid", UserPromptSubmit, "", map[string]any{"prompt": "hi"}); err != nil {
		t.Fatal(err)
	}
}
