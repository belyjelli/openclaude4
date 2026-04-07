package session

import sdk "github.com/sashabaranov/go-openai"

// DefaultCompactTail is the default number of non-system messages kept after compaction.
const DefaultCompactTail = 24

// CompactTail keeps the first system message (if present) and the last maxAfterSystem messages.
func CompactTail(msgs []sdk.ChatCompletionMessage, maxAfterSystem int) []sdk.ChatCompletionMessage {
	if len(msgs) == 0 {
		return nil
	}
	if maxAfterSystem < 1 {
		maxAfterSystem = 1
	}
	if len(msgs) > 0 && msgs[0].Role == sdk.ChatMessageRoleSystem {
		sys := msgs[0]
		rest := msgs[1:]
		if len(rest) <= maxAfterSystem {
			return msgs
		}
		out := make([]sdk.ChatCompletionMessage, 0, 1+maxAfterSystem)
		out = append(out, sys)
		out = append(out, rest[len(rest)-maxAfterSystem:]...)
		return out
	}
	if len(msgs) <= maxAfterSystem {
		return msgs
	}
	return msgs[len(msgs)-maxAfterSystem:]
}
