package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

const sideQuestionSystem = `You are a concise assistant. Answer the user's question directly in plain text.
Do not suggest running shell commands or using external tools unless the user explicitly asks.`

// SideQuestion runs a single non-tool completion (does not modify the main session transcript).
func SideQuestion(ctx context.Context, client StreamClient, question string) (string, error) {
	if client == nil {
		return "", errors.New("side question: no client")
	}
	q := strings.TrimSpace(question)
	if q == "" {
		return "", errors.New("usage: /btw <your question>")
	}
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: sideQuestionSystem},
		{Role: sdk.ChatMessageRoleUser, Content: q},
	}
	stream, err := client.StreamChatWithTools(ctx, msgs, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = stream.Close() }()

	assistant, err := consumeAssistantStream(stream, io.Discard, nil, 1, streamClientModel(client))
	if err != nil {
		return "", err
	}
	if len(assistant.ToolCalls) > 0 {
		return strings.TrimSpace(assistant.Content),
			fmt.Errorf("model requested tools in a side question (try rephrasing; content above may be partial)")
	}
	return strings.TrimSpace(assistant.Content), nil
}
