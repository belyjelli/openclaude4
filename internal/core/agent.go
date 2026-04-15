package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gitlawb/openclaude4/internal/apialign"
	"github.com/gitlawb/openclaude4/internal/bashv2"
	"github.com/gitlawb/openclaude4/internal/hooks"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

const defaultMaxIterations = 24

// StreamClient can start a streaming chat completion with tools.
type StreamClient interface {
	StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error)
}

// ConfirmTool is invoked before running a dangerous tool (after [Agent.PermissionPolicy], if any).
// The transport may set side effects on the outcome (session auto-approve, new allow rules) before returning.
type ConfirmTool func(toolName string, args map[string]any) PermissionOutcome

// Agent runs the multi-turn tool loop with streaming assistant text to out.
type Agent struct {
	Client   StreamClient
	Registry *tools.Registry
	Confirm  ConfirmTool
	// PermissionPolicy, if set, runs before prompting; if it returns decided=true, Confirm is skipped.
	PermissionPolicy PermissionPolicy
	Out              io.Writer
	MaxIterations    int
	// EventSubTaskDepth is stamped on every emitted [Event] (0 main thread; incremented for Task / skill-fork children).
	EventSubTaskDepth int
	// OnEvent receives structured kernel events (streaming text, tool calls, errors).
	// Optional; TUI / headless transports use this instead of scraping stdout.
	OnEvent EventHandler
}

func (a *Agent) emit(e Event) {
	if a != nil && a.OnEvent != nil {
		e.SubTaskDepth = a.EventSubTaskDepth
		a.OnEvent(RedactEventForLog(e))
	}
}

// DefaultSystemPrompt is prepended once at the start of the transcript (git/gh guidance aligns with OpenClaude v3).
const DefaultSystemPrompt = `You are OpenClaude v4, a terminal coding agent. Use tools when they help. ` +
	`Prefer reading files before editing. Keep shell commands short; respect the workspace boundary. ` +
	`Explain briefly what you are doing when using dangerous tools.

# Git and GitHub CLI (gh)
Use the gh command via the Bash tool for GitHub work: issues, pull requests, checks, releases, and API reads. ` +
	`If the user gives a github.com URL, prefer resolving it with gh (e.g. gh pr view, gh issue view, gh api) instead of guessing.

# Committing with git (when the user asks)
Git safety: do not change git config; avoid destructive commands (force push to main, reset --hard, etc.) unless explicitly requested; ` +
	`do not skip hooks (--no-verify, --no-gpg-sign) unless the user asks; prefer new commits over amend after a failed hook; ` +
	`stage specific paths instead of git add -A when possible; only commit when explicitly requested.
For the message, prefer a heredoc, e.g. git commit -m "$(cat <<'EOF'
Your message here.
EOF
)"

# Pull requests (when the user asks)
Before creating a PR: inspect git status, diff, remote tracking, and git log / git diff base...HEAD for the full branch story. ` +
	`Keep the title short; put detail in the body. Push with -u when needed. Create the PR with gh pr create, using a heredoc for the body, e.g.:
gh pr create --title "short title" --body "$(cat <<'EOF'
## Summary
- ...

## Test plan
- [ ] ...
EOF
)"
Return the PR URL when done.

# Other GitHub examples
- PR comments JSON: gh api repos/OWNER/REPO/pulls/123/comments
- Poll CI: gh run view / gh pr checks instead of arbitrary sleep when checking workflow state.
` + GitAndGitHubWorkflowInstructions

// RunUserTurn appends a text-only user message, then loops model→tools until the model responds without tool calls.
func (a *Agent) RunUserTurn(ctx context.Context, messages *[]sdk.ChatCompletionMessage, userText string) error {
	return a.runUserTurnWithUserMessage(ctx, messages, sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleUser,
		Content: userText,
	})
}

// RunUserTurnMulti appends a multimodal user message (e.g. text + image_url parts), then runs the same loop as [Agent.RunUserTurn].
func (a *Agent) RunUserTurnMulti(ctx context.Context, messages *[]sdk.ChatCompletionMessage, parts []sdk.ChatMessagePart) error {
	if len(parts) == 0 {
		return errors.New("agent: empty multimodal user message")
	}
	return a.runUserTurnWithUserMessage(ctx, messages, sdk.ChatCompletionMessage{
		Role:         sdk.ChatMessageRoleUser,
		MultiContent: parts,
	})
}

// RunUserTurnScoped is like [Agent.RunUserTurn] but restricts tools to allow (v3 allowed_tools).
// When allow is empty, all tools except Task are available (same as [Agent.SubAgentRegistry] with empty allow).
func (a *Agent) RunUserTurnScoped(ctx context.Context, messages *[]sdk.ChatCompletionMessage, userText string, allow []string) error {
	child := a.forkLikeAgent(a.SubAgentRegistry(allow))
	return child.runUserTurnWithUserMessage(ctx, messages, sdk.ChatCompletionMessage{
		Role:    sdk.ChatMessageRoleUser,
		Content: userText,
	})
}

// RunUserTurnMultiScoped is the multimodal variant of [Agent.RunUserTurnScoped].
func (a *Agent) RunUserTurnMultiScoped(ctx context.Context, messages *[]sdk.ChatCompletionMessage, parts []sdk.ChatMessagePart, allow []string) error {
	if len(parts) == 0 {
		return errors.New("agent: empty multimodal user message")
	}
	child := a.forkLikeAgent(a.SubAgentRegistry(allow))
	return child.runUserTurnWithUserMessage(ctx, messages, sdk.ChatCompletionMessage{
		Role:         sdk.ChatMessageRoleUser,
		MultiContent: parts,
	})
}

func (a *Agent) forkLikeAgent(reg *tools.Registry) *Agent {
	return &Agent{
		Client:            a.Client,
		Registry:          reg,
		Confirm:           a.Confirm,
		PermissionPolicy:  a.PermissionPolicy,
		Out:               a.Out,
		OnEvent:           a.OnEvent,
		MaxIterations:     a.MaxIterations,
		EventSubTaskDepth: a.EventSubTaskDepth,
	}
}

func (a *Agent) runUserTurnWithUserMessage(ctx context.Context, messages *[]sdk.ChatCompletionMessage, user sdk.ChatCompletionMessage) error {
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

	user.Role = sdk.ChatMessageRoleUser
	// v3: createUserMessage uses content || NO_CONTENT_MESSAGE (never omit user content).
	if user.Content == "" && len(user.MultiContent) == 0 {
		user.Content = apialign.NoContentUserMessage
	}
	*messages = append(*messages, user)
	a.emit(Event{Kind: KindUserMessage, UserText: UserMessageSummary(user)})
	dispatchSkillHook(ctx, hooks.UserPromptSubmit, map[string]any{
		"session_id": hooks.SessionIDFrom(ctx),
		"cwd":        tools.WorkDir(ctx),
		"prompt":     UserMessageSummary(user),
	})

	emit := func(e Event) { a.emit(e) }

	for i := range max {
		modelRound := i + 1
		stream, err := a.Client.StreamChatWithTools(ctx, *messages, oaiTools)
		if err != nil {
			a.emit(Event{Kind: KindError, Message: fmt.Sprintf("stream: %v", err)})
			*messages = (*messages)[:len(*messages)-1]
			return fmt.Errorf("stream: %w", err)
		}

		model := streamClientModel(a.Client)
		assistant, err := consumeAssistantStream(stream, a.Out, emit, modelRound, model)
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

			if tool.IsDangerous() {
				skipConfirm := false
				if name == "Bash" {
					if bs := bashv2.FromContext(ctx); bs != nil {
						gr := bs.Gate(ctx, tools.WorkDir(ctx), args)
						switch gr.Phase {
						case bashv2.PhaseDeny:
							msg := strings.TrimSpace(gr.Message)
							if msg == "" {
								msg = "Bash command rejected by validation or policy."
							}
							*messages = append(*messages, toolResultMessage(tc.ID, name, "", errors.New(msg)))
							a.emit(Event{
								Kind:           KindToolResult,
								ToolCallID:     tc.ID,
								ToolName:       name,
								ToolExecError:  msg,
								ToolResultText: "",
							})
							continue
						case bashv2.PhaseAllow:
							skipConfirm = true
						}
					}
				}
				if !skipConfirm {
					var permOutcome PermissionOutcome
					permHandled := false
					permReason := ""
					if a.PermissionPolicy != nil {
						if o, ok, tag := a.PermissionPolicy(name, args); ok {
							permOutcome = o
							permHandled = true
							permReason = tag
						}
					}
					if !permHandled && a.Confirm != nil {
						a.emit(Event{
							Kind:             KindPermissionPrompt,
							PermissionTool:   name,
							ToolArgs:         args,
							PermissionReason: "dangerous_tool",
						})
						permOutcome = a.Confirm(name, args)
						permHandled = true
						if permOutcome.Allow {
							permReason = "user_allow"
						} else {
							permReason = "user_deny"
						}
					}
					if permHandled {
						if permReason == "" {
							if permOutcome.Allow {
								permReason = "user_allow"
							} else {
								permReason = "user_deny"
							}
						}
						a.emit(Event{
							Kind:                         KindPermissionResult,
							PermissionTool:               name,
							PermissionApproved:           permOutcome.Allow,
							PermissionDeclineNote:        permOutcome.DeclineUserNote,
							PermissionRulesAdded:         append([]string(nil), permOutcome.AddAllowRules...),
							PermissionSessionAutoApprove: permOutcome.EnableSessionAutoApprove,
							PermissionReason:             permReason,
						})
						if !permOutcome.Allow {
							declined := DeclineToolMessage(permOutcome.DeclineUserNote)
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
				}
			}

			dispatchSkillHook(ctx, hooks.PreToolUse, map[string]any{
				"session_id": hooks.SessionIDFrom(ctx),
				"cwd":        tools.WorkDir(ctx),
				"tool_name":  name,
				"tool_input": args,
			})

			execCtx := tools.WithToolCallID(ctx, tc.ID)
			if a.OnEvent != nil {
				execCtx = bashv2.WithStreamHook(execCtx, func(toolCallID, chunk string, totalBytes int) {
					a.emit(Event{
						Kind:                 KindToolOutputDelta,
						ToolCallID:           toolCallID,
						TextChunk:            chunk,
						ToolOutputTotalBytes: totalBytes,
						ToolName:             name,
					})
				})
			}

			result, err := tool.Execute(execCtx, args)
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
			postPayload := map[string]any{
				"session_id":  hooks.SessionIDFrom(ctx),
				"cwd":         tools.WorkDir(ctx),
				"tool_name":   name,
				"tool_input":  args,
				"tool_result": result,
			}
			if err != nil {
				postPayload["tool_error"] = err.Error()
			}
			dispatchSkillHook(ctx, hooks.PostToolUse, postPayload)
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
	if content == "" {
		content = apialign.EmptyToolResultCompletedMessage(name)
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

// streamClientModel returns the configured model id when the client exposes Model() (e.g. *openaicomp.Client).
func streamClientModel(c StreamClient) string {
	type modeler interface {
		Model() string
	}
	if x, ok := c.(modeler); ok {
		return x.Model()
	}
	return ""
}

func consumeAssistantStream(stream *sdk.ChatCompletionStream, out io.Writer, emit func(Event), modelRound int, model string) (sdk.ChatCompletionMessage, error) {
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

	rawContent := content.String()
	toolCalls := flattenToolCalls(acc)
	assistantText := rawContent
	if len(toolCalls) == 0 && xmlToolFallbackEnabledForModel(model) {
		if extracted := extractXMLToolCallsFromContent(rawContent); len(extracted) > 0 {
			toolCalls = extracted
			assistantText = strings.TrimSpace(cleanToolCallMarkupFromContent(stripReasoningBlocks(rawContent)))
		}
	}

	msg := sdk.ChatCompletionMessage{
		Role:      sdk.ChatMessageRoleAssistant,
		Content:   assistantText,
		ToolCalls: toolCalls,
	}
	emit(Event{
		Kind:            KindAssistantFinished,
		AssistantText:   assistantText,
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
		Content: EffectiveSystemPrompt(),
	}
	*messages = append([]sdk.ChatCompletionMessage{sys}, *messages...)
}

func dispatchSkillHook(ctx context.Context, ev hooks.SkillHookEvent, payload map[string]any) {
	sid := hooks.SessionIDFrom(ctx)
	if sid == "" {
		return
	}
	_ = hooks.Default().Dispatch(ctx, sid, ev, "", payload)
}
