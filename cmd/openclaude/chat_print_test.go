package main

import (
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestLastAssistantReply(t *testing.T) {
	t.Parallel()
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "hi"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "first"},
		{Role: sdk.ChatMessageRoleTool, Content: "{}", ToolCallID: "1"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "final answer"},
	}
	if got := lastAssistantReply(msgs); got != "final answer" {
		t.Fatalf("got %q", got)
	}
}

func TestLastAssistantReply_None(t *testing.T) {
	t.Parallel()
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "only user"},
	}
	if got := lastAssistantReply(msgs); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}
