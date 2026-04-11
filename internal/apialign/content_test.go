package apialign

import (
	"encoding/json"
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestEmptyToolResultCompletedMessage(t *testing.T) {
	if got := EmptyToolResultCompletedMessage("Bash"); got != "(Bash completed with no output)" {
		t.Fatalf("got %q", got)
	}
	if got := EmptyToolResultCompletedMessage(""); got != "(tool completed with no output)" {
		t.Fatalf("got %q", got)
	}
}

func TestTranscript_toolJSONHasContentKey(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "hi"},
		{
			Role: sdk.ChatMessageRoleAssistant,
			ToolCalls: []sdk.ToolCall{{
				ID:   "call_1",
				Type: sdk.ToolTypeFunction,
				Function: sdk.FunctionCall{
					Name:      "Read",
					Arguments: `{}`,
				},
			}},
		},
		{Role: sdk.ChatMessageRoleTool, Name: "Read", ToolCallID: "call_1", Content: ""},
	}
	Transcript(msgs)
	if msgs[2].Content != "(Read completed with no output)" {
		t.Fatalf("tool content: %q", msgs[2].Content)
	}
	raw, err := json.Marshal(msgs[2])
	if err != nil {
		t.Fatal(err)
	}
	var dec map[string]any
	if err := json.Unmarshal(raw, &dec); err != nil {
		t.Fatal(err)
	}
	if _, ok := dec["content"]; !ok {
		t.Fatalf("missing content key in %s", raw)
	}
}

func TestTranscript_userSystemAssistant(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: ""},
		{Role: sdk.ChatMessageRoleUser, Content: ""},
		{Role: sdk.ChatMessageRoleAssistant, Content: ""},
		{Role: sdk.ChatMessageRoleAssistant, Content: ""},
	}
	Transcript(msgs)
	if msgs[0].Content != NoContentUserMessage || msgs[1].Content != NoContentUserMessage {
		t.Fatalf("system/user: %#v %#v", msgs[0], msgs[1])
	}
	if msgs[2].Content != NoContentUserMessage {
		t.Fatalf("non-final assistant: %q", msgs[2].Content)
	}
	if msgs[3].Content != "" {
		t.Fatalf("final assistant should stay empty, got %q", msgs[3].Content)
	}
}
