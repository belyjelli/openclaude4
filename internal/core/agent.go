package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

const defaultMaxIterations = 24

// StreamClient can start a streaming chat completion with tools.
type StreamClient interface {
	StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error)
}

// ConfirmTool is invoked before running a dangerous tool; return false to skip.
type ConfirmTool func(toolName string, args map[string]any) bool

// Agent runs the multi-turn tool loop with streaming assistant text to out.
type Agent struct {
	Client        StreamClient
	Registry      *tools.Registry
	Confirm       ConfirmTool
	Out           io.Writer
	MaxIterations int
	// OnEvent receives structured kernel events (streaming text, tool calls, errors).
	// Optional; TUI / headless transports use this instead of scraping stdout.
	OnEvent EventHandler
}

func (a *Agent) emit(e Event) {
	if a != nil && a.OnEvent != nil {
		a.OnEvent(e)
	}
}

// DefaultSystemPrompt is prepended once at the start of the transcript.
const DefaultSystemPrompt = `You are OpenClaude v4, a terminal coding agent. Use tools when they help. ` +
	`Prefer reading files before editing. Keep shell commands short; respect the workspace boundary. ` +
	`Explain briefly what you are doing when using dangerous tools.`

// RunUserTurn appends the user message, then loops model→tools until the model responds without tool calls.
func (a *Agent) RunUserTurn(ctx context.Context, messages *[]sdk.ChatCompletionMessage, userText string) error {
	if a.Client == nil || a.Registry == nil {
		return errors.New("agent: missing client or registry")
	}
	if a.Out == nil {
		a.Out = io.Discard
	}
	max := a.MaxIterations
	if max <= 0 {
		max = defaultMaxIterations
	}

	ensureSystemMessage(messages)

	oaiTools, err := tools.OpenAITools(a.Registry)
	if err != nil {
		a.emit(Event{Kind: KindError, Message: fmt.Sprintf("openai tools: %v", err)})
		return fmt.Errorf("openai tools: %w", err)
	}

	*messages = append(*messages, sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleUser,
		Content: userText,
	})
	a.emit(Event{Kind: KindUserMessage, UserText: userText})

	emit := func(e Event) { a.emit(e) }

	for i := range max {
		modelRound := i + 1
		stream, err := a.Client.StreamChatWithTools(ctx, *messages, oaiTools)
		if err != nil {
			a.emit(Event{Kind: KindError, Message: fmt.Sprintf("stream: %v", err)})
			*messages = (*messages)[:len(*messages)-1]
			return fmt.Errorf("stream: %w", err)
		}

		assistant, err := consumeAssistantStream(stream, a.Out, emit, modelRound)
		_ = stream.Close()
		if err != nil {
			*messages = (*messages)[:len(*messages)-1]
			return err
		}

		*messages = append(*messages, assistant)

		if len(assistant.ToolCalls) == 0 {
			a.emit(Event{Kind: KindTurnComplete})
			return nil
		}

		for _, tc := range assistant.ToolCalls {
			name := tc.Function.Name
			args, err := parseToolArgs(tc.Function.Arguments)
			a.emit(Event{
				Kind:         KindToolCall,
				ToolCallID:   tc.ID,
				ToolName:     name,
				ToolArgs:     args,
				ToolArgsJSON: tc.Function.Arguments,
			})
			if err != nil {
				*messages = append(*messages, toolResultMessage(tc.ID, name, "", err))
				a.emit(Event{
					Kind:           KindToolResult,
					ToolCallID:     tc.ID,
					ToolName:       name,
					ToolExecError:  err.Error(),
					ToolResultText: "",
				})
				continue
			}

			tool, ok := a.Registry.Get(name)
			if !ok {
				e := fmt.Errorf("unknown tool %q", name)
				*messages = append(*messages, toolResultMessage(tc.ID, name, "", e))
				a.emit(Event{
					Kind:           KindToolResult,
					ToolCallID:     tc.ID,
					ToolName:       name,
					ToolExecError:  e.Error(),
					ToolResultText: "",
				})
				continue
			}

			if tool.IsDangerous() && a.Confirm != nil {
				a.emit(Event{Kind: KindPermissionPrompt, PermissionTool: name, ToolArgs: args})
				ok := a.Confirm(name, args)
				a.emit(Event{Kind: KindPermissionResult, PermissionTool: name, PermissionApproved: ok})
				if !ok {
					const declined = "User declined this tool execution."
					*messages = append(*messages, toolResultMessage(tc.ID, name, declined, nil))
					a.emit(Event{
						Kind:           KindToolResult,
						ToolCallID:     tc.ID,
						ToolName:       name,
						ToolResultText: declined,
					})
					continue
				}
			}

			result, err := tool.Execute(ctx, args)
			*messages = append(*messages, toolResultMessage(tc.ID, name, result, err))
			ev := Event{
				Kind:           KindToolResult,
				ToolCallID:     tc.ID,
				ToolName:       name,
				ToolResultText: result,
			}
			if err != nil {
				ev.ToolExecError = err.Error()
			}
			a.emit(ev)
		}
	}

	a.emit(Event{Kind: KindError, Message: fmt.Sprintf("agent: exceeded %d tool iterations", max)})
	*messages = (*messages)[:len(*messages)-1]
	return fmt.Errorf("agent: exceeded %d tool iterations", max)
}

func toolResultMessage(toolCallID, name, result string, execErr error) sdk.ChatCompletionMessage {
	content := result
	if execErr != nil {
		if content != "" {
			content += "\n"
		}
		content += "Error: " + execErr.Error()
	}
	return sdk.ChatCompletionMessage{
		Role:       sdk.ChatMessageRoleTool,
		Name:       name,
		ToolCallID: toolCallID,
		Content:    content,
	}
}

func parseToolArgs(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("tool arguments JSON: %w", err)
	}
	return m, nil
}

type toolCallAcc struct {
	id   string
	typ  string
	name strings.Builder
	args strings.Builder
}

func consumeAssistantStream(stream *sdk.ChatCompletionStream, out io.Writer, emit func(Event), modelRound int) (sdk.ChatCompletionMessage, error) {
	if emit == nil {
		emit = func(Event) {}
	}
	var content strings.Builder
	acc := make(map[int]*toolCallAcc)
	var finish sdk.FinishReason

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			emit(Event{Kind: KindError, Message: err.Error(), AssistantRounds: modelRound})
			return sdk.ChatCompletionMessage{}, err
		}
		if len(resp.Choices) == 0 {
			continue
		}
		ch := resp.Choices[0]
		if ch.FinishReason != "" {
			finish = ch.FinishReason
		}
		delta := ch.Delta
		if delta.Refusal != "" {
			emit(Event{Kind: KindModelRefusal, Message: delta.Refusal, AssistantRounds: modelRound})
			return sdk.ChatCompletionMessage{}, fmt.Errorf("model refusal: %s", delta.Refusal)
		}
		if delta.Content != "" {
			_, _ = out.Write([]byte(delta.Content))
			content.WriteString(delta.Content)
			emit(Event{
				Kind:            KindAssistantTextDelta,
				TextChunk:       delta.Content,
				AssistantRounds: modelRound,
			})
		}
		for _, tc := range delta.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			b := acc[idx]
			if b == nil {
				b = &toolCallAcc{}
				acc[idx] = b
			}
			if tc.ID != "" {
				b.id = tc.ID
			}
			if tc.Type != "" {
				b.typ = string(tc.Type)
			}
			if tc.Function.Name != "" {
				b.name.WriteString(tc.Function.Name)
			}
			if tc.Function.Arguments != "" {
				b.args.WriteString(tc.Function.Arguments)
			}
		}
	}

	_, _ = out.Write([]byte("\n"))

	toolCalls := flattenToolCalls(acc)
	msg := sdk.ChatCompletionMessage{
		Role:      sdk.ChatMessageRoleAssistant,
		Content:   content.String(),
		ToolCalls: toolCalls,
	}
	emit(Event{
		Kind:            KindAssistantFinished,
		AssistantText:   content.String(),
		ToolCallCount:   len(toolCalls),
		FinishReason:    string(finish),
		AssistantRounds: modelRound,
	})
	return msg, nil
}

func flattenToolCalls(acc map[int]*toolCallAcc) []sdk.ToolCall {
	if len(acc) == 0 {
		return nil
	}
	idxs := make([]int, 0, len(acc))
	for k := range acc {
		idxs = append(idxs, k)
	}
	sort.Ints(idxs)
	out := make([]sdk.ToolCall, 0, len(idxs))
	for _, i := range idxs {
		b := acc[i]
		typ := sdk.ToolType(b.typ)
		if typ == "" {
			typ = sdk.ToolTypeFunction
		}
		out = append(out, sdk.ToolCall{
			ID:   b.id,
			Type: typ,
			Function: sdk.FunctionCall{
				Name:      b.name.String(),
				Arguments: b.args.String(),
			},
		})
	}
	return out
}

func ensureSystemMessage(messages *[]sdk.ChatCompletionMessage) {
	if len(*messages) > 0 && (*messages)[0].Role == sdk.ChatMessageRoleSystem {
		return
	}
	sys := sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleSystem,
		Content: DefaultSystemPrompt,
	}
	*messages = append([]sdk.ChatCompletionMessage{sys}, *messages...)
}
