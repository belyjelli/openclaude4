package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

// StreamCompleter matches [core.StreamClient] without importing core (avoids test import cycles).
type StreamCompleter interface {
	StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error)
}

// RoughTokenEstimate is a fast heuristic (~4 chars per token) for budgeting; not exact.
func RoughTokenEstimate(msgs []sdk.ChatCompletionMessage) int {
	var n int
	for _, m := range msgs {
		n += roughStringTokens(m.Content)
		for _, tc := range m.ToolCalls {
			n += roughStringTokens(tc.ID)
			n += roughStringTokens(string(tc.Type))
			n += roughStringTokens(tc.Function.Name)
			n += roughStringTokens(tc.Function.Arguments)
		}
		if m.Role == sdk.ChatMessageRoleTool {
			n += roughStringTokens(m.ToolCallID)
			n += roughStringTokens(m.Name)
		}
	}
	return n
}

func roughStringTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}

// ApplyTokenThreshold compacts or (optionally) summarizes the transcript when the rough
// token estimate exceeds threshold. threshold <= 0 disables the check.
// systemPrompt is used when summarization resets the transcript (typically [core.EffectiveSystemPrompt]).
func ApplyTokenThreshold(ctx context.Context, client StreamCompleter, messages *[]sdk.ChatCompletionMessage, threshold int, summarize bool, tail int, systemPrompt string) error {
	if messages == nil || threshold <= 0 {
		return nil
	}
	msgs := *messages
	if RoughTokenEstimate(msgs) <= threshold {
		return nil
	}
	if tail <= 0 {
		tail = DefaultCompactTail
	}
	if summarize && client != nil {
		sum, err := summarizeTranscript(ctx, client, msgs)
		if err == nil && sum != "" {
			if systemPrompt == "" {
				systemPrompt = "You are a coding assistant."
			}
			*messages = []sdk.ChatCompletionMessage{
				{Role: sdk.ChatMessageRoleSystem, Content: systemPrompt},
				{
					Role:    sdk.ChatMessageRoleUser,
					Content: "Summary of earlier conversation (auto-compacted over token limit):\n\n" + sum,
				},
			}
			return nil
		}
	}
	*messages = CompactTail(msgs, tail)
	return nil
}

func summarizeTranscript(ctx context.Context, client StreamCompleter, msgs []sdk.ChatCompletionMessage) (string, error) {
	if client == nil {
		return "", errors.New("session: no client for summarize")
	}
	raw, err := json.Marshal(msgs)
	if err != nil {
		return "", err
	}
	const maxIn = 120_000
	if len(raw) > maxIn {
		raw = raw[:maxIn]
	}
	prompt := []sdk.ChatCompletionMessage{
		{
			Role: sdk.ChatMessageRoleSystem,
			Content: "You summarize a coding-agent chat transcript. Output concise plain text: key actions, files, errors, open tasks. " +
				"No markdown code fences, no preamble.",
		},
		{Role: sdk.ChatMessageRoleUser, Content: "Transcript (JSON):\n" + string(raw)},
	}
	stream, err := client.StreamChatWithTools(ctx, prompt, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = stream.Close() }()
	return drainSummaryStream(stream)
}

func drainSummaryStream(stream *sdk.ChatCompletionStream) (string, error) {
	var b strings.Builder
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			continue
		}
		d := resp.Choices[0].Delta
		if d.Refusal != "" {
			return "", fmt.Errorf("model refusal: %s", d.Refusal)
		}
		if len(d.ToolCalls) > 0 {
			return "", errors.New("unexpected tool calls in summary stream")
		}
		if d.Content != "" {
			b.WriteString(d.Content)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
