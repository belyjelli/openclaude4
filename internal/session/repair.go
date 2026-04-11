package session

import (
	"github.com/gitlawb/openclaude4/internal/apialign"
	sdk "github.com/sashabaranov/go-openai"
)

const recoveryToolResultText = "[session recovery] Tool run was interrupted or incomplete before a result was saved."

// RepairTranscript ensures every assistant message with tool calls has matching tool-role
// messages for each tool_call_id. Missing results are appended so the transcript is valid
// for the chat API after an interrupted save or crash mid-tool round.
func RepairTranscript(msgs []sdk.ChatCompletionMessage) []sdk.ChatCompletionMessage {
	if len(msgs) == 0 {
		return msgs
	}
	out := append([]sdk.ChatCompletionMessage(nil), msgs...)
	for i := 0; i < len(out); {
		if out[i].Role != sdk.ChatMessageRoleAssistant || len(out[i].ToolCalls) == 0 {
			i++
			continue
		}
		start := i + 1
		end := start
		seen := make(map[string]struct{})
		for end < len(out) && out[end].Role == sdk.ChatMessageRoleTool {
			if out[end].ToolCallID != "" {
				seen[out[end].ToolCallID] = struct{}{}
			}
			end++
		}
		var insert []sdk.ChatCompletionMessage
		for _, tc := range out[i].ToolCalls {
			id := tc.ID
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			name := tc.Function.Name
			insert = append(insert, sdk.ChatCompletionMessage{
				Role:       sdk.ChatMessageRoleTool,
				Name:       name,
				ToolCallID: id,
				Content:    recoveryToolResultText,
			})
		}
		if len(insert) == 0 {
			i++
			continue
		}
		out = append(out[:end], append(insert, out[end:]...)...)
		i = end + len(insert)
	}
	apialign.Transcript(out)
	return out
}
