package providers

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

// ChatStreamer streams assistant tokens for a fixed message list (Phase 0: no tools).
type ChatStreamer interface {
	StreamChat(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error)
}
