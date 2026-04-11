package core

import (
	"context"
	"fmt"
	"io"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// RunSkillForked runs expanded skill instructions in an isolated agent loop (v3 context:fork).
func RunSkillForked(ctx context.Context, parent *Agent, allow []string, expandedBody string, maxIter int) (string, error) {
	if parent == nil {
		return "", fmt.Errorf("skill fork: nil agent")
	}
	subMax := maxIter
	if subMax <= 0 {
		subMax = defaultTaskSubMaxIter
	}
	if subMax > defaultMaxIterations {
		subMax = defaultMaxIterations
	}
	reg := parent.SubAgentRegistry(allow)
	child := &Agent{
		Client:           parent.Client,
		Registry:         reg,
		Confirm:          parent.Confirm,
		PermissionPolicy: parent.PermissionPolicy,
		Out:              io.Discard,
		OnEvent:          parent.OnEvent,
		MaxIterations:    subMax,
	}
	sys := SubTaskSystemPrompt + "\n\n--- Skill ---\n\n" + expandedBody
	var subMsgs []sdk.ChatCompletionMessage
	subMsgs = append(subMsgs, sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleSystem,
		Content: sys,
	})
	if err := child.RunUserTurn(ctx, &subMsgs, "Execute the skill using tools as needed. When finished, reply with a concise summary."); err != nil {
		return "", err
	}
	out := lastNonToolAssistantContent(subMsgs)
	if out == "" {
		return "(forked skill finished with no assistant text)", nil
	}
	return out, nil
}

// SubAgentRegistry builds a tool registry for a sub-agent: optional allow-list, always omits Task.
func (a *Agent) SubAgentRegistry(allow []string) *tools.Registry {
	if a == nil || a.Registry == nil {
		return tools.NewRegistry()
	}
	base := tools.CloneRegistryAllow(a.Registry, allow)
	return tools.CloneRegistryOmit(base, "Task")
}
