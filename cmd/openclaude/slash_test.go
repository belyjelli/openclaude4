package main

import (
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestCompactTail_SystemAndRest(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: "sys"},
		{Role: sdk.ChatMessageRoleUser, Content: "u1"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "a1"},
		{Role: sdk.ChatMessageRoleUser, Content: "u2"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "a2"},
		{Role: sdk.ChatMessageRoleUser, Content: "u3"},
	}
	out := compactTail(msgs, 2)
	if len(out) != 3 {
		t.Fatalf("len %d want 3: %+v", len(out), out)
	}
	if out[0].Content != "sys" || out[1].Content != "a2" || out[2].Content != "u3" {
		t.Fatalf("wrong tail: %+v", out)
	}
}

func TestCompactTail_NoSystem(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "a"},
		{Role: sdk.ChatMessageRoleUser, Content: "b"},
		{Role: sdk.ChatMessageRoleUser, Content: "c"},
	}
	out := compactTail(msgs, 2)
	if len(out) != 2 || out[0].Content != "b" || out[1].Content != "c" {
		t.Fatalf("got %+v", out)
	}
}
