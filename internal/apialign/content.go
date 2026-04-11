package apialign

import (
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

// NoContentUserMessage matches openclaude3 NO_CONTENT_MESSAGE (src/constants/messages.ts).
const NoContentUserMessage = "(no content)"

// EmptyToolResultCompletedMessage matches the placeholder openclaude3 injects for empty
// tool_result content in maybePersistLargeToolResult (src/utils/toolResultStorage.ts).
func EmptyToolResultCompletedMessage(toolName string) string {
	t := strings.TrimSpace(toolName)
	if t == "" {
		t = "tool"
	}
	return "(" + t + " completed with no output)"
}

// Transcript ensures user, tool, system, and non-final assistant messages carry non-empty
// string content when serialized (go-openai omits json "content" for "").
// This matches OpenClaude v3 behavior and avoids Anthropic 400s when using OpenRouter
// or other gateways that enforce the Messages API shape.
func Transcript(msgs []sdk.ChatCompletionMessage) {
	for i := range msgs {
		m := &msgs[i]
		switch m.Role {
		case sdk.ChatMessageRoleSystem:
			if m.Content == "" && len(m.MultiContent) == 0 {
				m.Content = NoContentUserMessage
			}
		case sdk.ChatMessageRoleUser:
			if m.Content == "" && len(m.MultiContent) == 0 {
				m.Content = NoContentUserMessage
			}
		case sdk.ChatMessageRoleTool:
			if m.Content == "" {
				m.Content = EmptyToolResultCompletedMessage(m.Name)
			}
		case sdk.ChatMessageRoleAssistant:
			// v3 allows an empty final assistant message (prefill). Others must be non-empty.
			if i == len(msgs)-1 {
				continue
			}
			if m.Content == "" && len(m.ToolCalls) == 0 {
				m.Content = NoContentUserMessage
			}
		}
	}
}
