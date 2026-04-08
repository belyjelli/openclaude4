package core

import (
	"fmt"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

// UserMessageSummary returns a short line for kernel events and logs (never includes raw image data).
func UserMessageSummary(msg sdk.ChatCompletionMessage) string {
	if msg.Role != sdk.ChatMessageRoleUser {
		return ""
	}
	if msg.Content != "" && len(msg.MultiContent) == 0 {
		return msg.Content
	}
	var text strings.Builder
	nImg := 0
	for _, p := range msg.MultiContent {
		switch p.Type {
		case sdk.ChatMessagePartTypeText:
			if strings.TrimSpace(p.Text) != "" {
				if text.Len() > 0 {
					text.WriteByte(' ')
				}
				text.WriteString(strings.TrimSpace(p.Text))
			}
		case sdk.ChatMessagePartTypeImageURL:
			if p.ImageURL != nil && strings.TrimSpace(p.ImageURL.URL) != "" {
				nImg++
			}
		}
	}
	out := strings.TrimSpace(text.String())
	if nImg > 0 {
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("(%d image(s))", nImg)
	}
	if out == "" {
		return "(empty user message)"
	}
	return out
}
