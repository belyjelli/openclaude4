// Package subtask implements the Task tool: a bounded sub-session with the same provider
// and tools as the parent, but a fresh transcript (no nested Task in the child registry).
package subtask

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

const taskToolName = "Task"

// Register adds the Task tool to reg. Call after all other tools (including MCP) are registered.
// confirm must match the parent agent's dangerous-tool policy.
func Register(reg *tools.Registry, client core.StreamClient, confirm core.ConfirmTool) {
	if reg == nil || client == nil {
		return
	}
	reg.Register(taskTool{
		client:  client,
		parent:  reg,
		confirm: confirm,
	})
}

type taskTool struct {
	client  core.StreamClient
	parent  *tools.Registry
	confirm core.ConfirmTool
}

func (taskTool) Name() string      { return taskToolName }
func (taskTool) IsDangerous() bool { return true }

func (taskTool) Description() string {
	return "Run a focused sub-task in a fresh conversation with the same tools (except Task). " +
		"Use for multi-step work that should not pollute the main transcript. " +
		"Returns the final assistant message text from the sub-session."
}

func (taskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal": map[string]any{
				"type":        "string",
				"description": "What the sub-agent should accomplish (be specific)",
			},
			"max_iterations": map[string]any{
				"type":        "number",
				"description": "Max model↔tool rounds in the sub-session (default 8, max 16)",
			},
		},
		"required": []string{"goal"},
	}
}

func (t taskTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	goal, _ := args["goal"].(string)
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "", fmt.Errorf("goal is required")
	}
	max := 8
	if v, ok := args["max_iterations"].(float64); ok {
		max = int(v)
	}
	if max < 1 {
		max = 1
	}
	if max > 16 {
		max = 16
	}

	child := cloneRegistryOmit(t.parent, taskToolName)
	sub := &core.Agent{
		Client:        t.client,
		Registry:      child,
		Confirm:       t.confirm,
		Out:           io.Discard,
		MaxIterations: max,
	}

	var msgs []sdk.ChatCompletionMessage
	if err := sub.RunUserTurn(ctx, &msgs, goal); err != nil {
		return "", fmt.Errorf("sub-task: %w", err)
	}

	last := lastAssistantText(msgs)
	if strings.TrimSpace(last) == "" {
		return "Task completed (sub-session produced no plain-text assistant content; it may have used tools only).", nil
	}
	return "Task completed. Final assistant message from sub-session:\n\n" + strings.TrimSpace(last), nil
}

func cloneRegistryOmit(src *tools.Registry, omit string) *tools.Registry {
	out := tools.NewRegistry()
	for _, tool := range src.List() {
		if tool.Name() == omit {
			continue
		}
		out.Register(tool)
	}
	return out
}

func lastAssistantText(msgs []sdk.ChatCompletionMessage) string {
	var last string
	for _, m := range msgs {
		if m.Role == sdk.ChatMessageRoleAssistant && strings.TrimSpace(m.Content) != "" {
			last = m.Content
		}
	}
	return last
}
