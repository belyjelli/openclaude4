package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// SubTaskSystemPrompt is used as the system message for an isolated Task sub-agent run.
const SubTaskSystemPrompt = `You are a sub-agent inside OpenClaude v4. Complete the assigned goal using tools when they help.
Prefer reading before editing. Stay within the workspace. When done, reply with a concise summary for the main agent.`

const defaultTaskSubMaxIter = 12

// TaskTool runs a short nested agent loop (same tools and client) and returns the sub-agent's final text.
type TaskTool struct {
	resolveAgent func() *Agent
	maxDepth     int
	subMaxIter   int
}

// NewTaskTool registers a lazy link to the parent agent (set after construction).
// maxDepth is how many nested Task levels are allowed (default 1 = no Task inside Task).
func NewTaskTool(resolveAgent func() *Agent) *TaskTool {
	return &TaskTool{
		resolveAgent: resolveAgent,
		maxDepth:     1,
		subMaxIter:   defaultTaskSubMaxIter,
	}
}

func (TaskTool) Name() string { return "Task" }

func (TaskTool) IsDangerous() bool { return true }

func (TaskTool) Description() string {
	return "Spawn a focused sub-agent with the same tools to complete a sub-goal. Returns the sub-agent's final text summary. Nested Task calls are limited by depth."
}

func (TaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal": map[string]any{
				"type":        "string",
				"description": "What the sub-agent should accomplish",
			},
			"context": map[string]any{
				"type":        "string",
				"description": "Optional extra constraints or background for the sub-agent",
			},
			"max_iterations": map[string]any{
				"type":        "number",
				"description": "Max model↔tool rounds for the sub-agent (default 12, max 24)",
			},
		},
		"required": []string{"goal"},
	}
}

func (t *TaskTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	parent := t.resolveAgent()
	if parent == nil || parent.Client == nil || parent.Registry == nil {
		return "", errors.New("task: agent not ready")
	}
	depth := tools.SubTaskDepth(ctx)
	if t.maxDepth <= 0 {
		t.maxDepth = 1
	}
	if depth >= t.maxDepth {
		return "", fmt.Errorf("task: max nested depth is %d (no further Task calls allowed here)", t.maxDepth)
	}
	goal := strings.TrimSpace(fmt.Sprint(args["goal"]))
	if goal == "" {
		return "", errors.New("task: goal is required")
	}
	extra := strings.TrimSpace(fmt.Sprint(args["context"]))
	subMax := t.subMaxIter
	if subMax <= 0 {
		subMax = defaultTaskSubMaxIter
	}
	if v, ok := args["max_iterations"].(float64); ok && v > 0 && v < 1000 {
		subMax = int(v)
	}
	if subMax > defaultMaxIterations {
		subMax = defaultMaxIterations
	}

	childCtx := tools.WithSubTaskDepth(ctx, depth+1)

	sub := &Agent{
		Client:        parent.Client,
		Registry:      parent.Registry,
		Confirm:       parent.Confirm,
		Out:           io.Discard,
		MaxIterations: subMax,
		OnEvent:       nil,
	}

	userText := goal
	if extra != "" {
		userText = goal + "\n\nAdditional context:\n" + extra
	}

	var subMsgs []sdk.ChatCompletionMessage
	subMsgs = append(subMsgs, sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleSystem,
		Content: SubTaskSystemPrompt,
	})

	if err := sub.RunUserTurn(childCtx, &subMsgs, userText); err != nil {
		return "", err
	}
	out := lastNonToolAssistantContent(subMsgs)
	if strings.TrimSpace(out) == "" {
		return "(sub-task finished with no assistant text)", nil
	}
	return out, nil
}

func lastNonToolAssistantContent(msgs []sdk.ChatCompletionMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == sdk.ChatMessageRoleAssistant && len(m.ToolCalls) == 0 {
			return m.Content
		}
	}
	return ""
}
