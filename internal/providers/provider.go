package providers

import (
	"context"

	sdk "github.com/sashabaranov/go-openai"
)

// ChatStreamer streams assistant text without tools (optional transports / tests).
type ChatStreamer interface {
	StreamChat(ctx context.Context, messages []sdk.ChatCompletionMessage) (*sdk.ChatCompletionStream, error)
}
