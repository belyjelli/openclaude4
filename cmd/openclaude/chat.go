package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
	sdk "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

func runChat(cmd *cobra.Command, _ []string) error {
	client, err := openaicomp.New()
	if err != nil {
		if errors.Is(err, openaicomp.ErrMissingAPIKey) {
			_, _ = fmt.Fprintln(os.Stderr, "Error: set OPENAI_API_KEY in your environment.")
			return err
		}
		return err
	}

	_, _ = fmt.Fprintf(os.Stderr, "OpenClaude v4 (phase 0). Model: %s. Type /help for commands. Ctrl+D to exit.\n", client.Model())

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var messages []sdk.ChatCompletionMessage
	reader := bufio.NewReader(os.Stdin)

	for {
		_, _ = fmt.Fprint(os.Stdout, "> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				_, _ = fmt.Fprintln(os.Stdout)
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case line == "/exit" || line == "/quit":
			return nil
		case line == "/help":
			printChatHelp()
			continue
		case line == "/clear":
			messages = nil
			_, _ = fmt.Fprintln(os.Stdout, "(conversation cleared)")
			continue
		case line == "/provider":
			printProviderInfo(client)
			continue
		case strings.HasPrefix(line, "/"):
			_, _ = fmt.Fprintf(os.Stderr, "Unknown command %q. Try /help.\n", strings.Fields(line)[0])
			continue
		}

		messages = append(messages, sdk.ChatCompletionMessage{
			Role:    sdk.ChatMessageRoleUser,
			Content: line,
		})

		stream, err := client.StreamChat(ctx, messages)
		if err != nil {
			// Drop the last user message so the user can retry.
			messages = messages[:len(messages)-1]
			return fmt.Errorf("stream start: %w", err)
		}

		var assistant strings.Builder
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				_ = stream.Close()
				messages = messages[:len(messages)-1]
				return fmt.Errorf("stream recv: %w", err)
			}
			if len(resp.Choices) == 0 {
				continue
			}
			delta := resp.Choices[0].Delta.Content
			_, _ = fmt.Fprint(os.Stdout, delta)
			assistant.WriteString(delta)
		}
		_ = stream.Close()
		_, _ = fmt.Fprintln(os.Stdout)

		text := assistant.String()
		if text != "" {
			messages = append(messages, sdk.ChatCompletionMessage{
				Role:    sdk.ChatMessageRoleAssistant,
				Content: text,
			})
		}
	}
}

func printProviderInfo(c *openaicomp.Client) {
	base := c.BaseURL()
	if base == "" {
		base = "(default OpenAI API URL)"
	}
	_, _ = fmt.Fprintf(os.Stdout, "Provider: OpenAI-compatible (go-openai)\nModel:   %s\nBase:    %s\nAPI key: %s\n",
		c.Model(), base, c.RedactedAPIKeySummary())
}

func printChatHelp() {
	const help = `Commands:
  /provider  Show active model, base URL, and redacted API key hint
  /clear     Clear conversation history for this session
  /help      Show this help
  /exit      Exit (same as /quit)
  /quit      Exit
`
	_, _ = fmt.Fprint(os.Stdout, help)
}
