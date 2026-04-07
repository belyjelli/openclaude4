// Package ocrpc implements the OpenClaude v4 gRPC API over the shared kernel ([core.Agent]).
package ocrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/grpc/openclaudev4"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Kernel wires the same [core.StreamClient] and [tools.Registry] types as the CLI/TUI.
type Kernel struct {
	Client      core.StreamClient
	Registry    *tools.Registry
	AutoApprove bool
}

// AgentService implements [openclaudev4.AgentServiceServer].
type AgentService struct {
	openclaudev4.UnimplementedAgentServiceServer
	Kernel Kernel
}

// Register mounts AgentService on the given gRPC registrar (typically [*grpc.Server]).
func Register(s grpc.ServiceRegistrar, k Kernel) {
	openclaudev4.RegisterAgentServiceServer(s, &AgentService{Kernel: k})
}

// Chat implements the bidirectional chat stream.
func (s *AgentService) Chat(stream grpc.BidiStreamingServer[openclaudev4.ClientMessage, openclaudev4.ServerMessage]) error {
	if s.Kernel.Client == nil || s.Kernel.Registry == nil {
		return status.Error(codes.FailedPrecondition, "kernel: missing client or registry")
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	clientCh := make(chan *openclaudev4.ClientMessage, 16)
	recvDone := make(chan struct{})
	go func() {
		defer close(clientCh)
		for {
			m, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case clientCh <- m:
			case <-ctx.Done():
				return
			}
		}
	}()
	close(recvDone) // reserved if we need recv err later
	_ = recvDone

	var sendMu sync.Mutex
	var sendErr error
	send := func(msg *openclaudev4.ServerMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		if sendErr != nil {
			return sendErr
		}
		if err := stream.Send(msg); err != nil {
			sendErr = err
			cancel()
			return err
		}
		return nil
	}

	var messages []sdk.ChatCompletionMessage
	var turnMu sync.Mutex
	var turnWG sync.WaitGroup
	pgate := new(permGate)

	for {
		select {
		case <-ctx.Done():
			turnWG.Wait()
			if sendErr != nil {
				return sendErr
			}
			return ctx.Err()
		case m, ok := <-clientCh:
			if !ok {
				turnWG.Wait()
				if sendErr != nil {
					return sendErr
				}
				return nil
			}
			if m == nil {
				continue
			}
			switch p := m.Payload.(type) {
			case *openclaudev4.ClientMessage_ChatRequest:
				req := p.ChatRequest
				if req == nil || strings.TrimSpace(req.GetUserText()) == "" {
					_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
						Message: "chat_request.user_text is required",
						Code:    "invalid_argument",
					}}})
					continue
				}
				turnWG.Add(1)
				go func(req *openclaudev4.ChatRequest) {
					defer turnWG.Done()
					turnMu.Lock()
					defer turnMu.Unlock()
					turnCtx := ctx
					if wd := strings.TrimSpace(req.GetWorkingDirectory()); wd != "" {
						turnCtx = tools.WithWorkDir(turnCtx, wd)
					}
					agent := &core.Agent{
						Client:   s.Kernel.Client,
						Registry: s.Kernel.Registry,
						Out:      io.Discard,
						OnEvent: func(e core.Event) {
							if msg := serverMessageFromEvent(e); msg != nil {
								_ = send(msg)
							}
						},
						Confirm: func(toolName string, args map[string]any) bool {
							if s.Kernel.AutoApprove {
								return true
							}
							return pgate.wait(ctx, send, toolName, args)
						},
					}
					_ = agent.RunUserTurn(turnCtx, &messages, req.GetUserText())
				}(req)

			case *openclaudev4.ClientMessage_UserInput:
				ui := p.UserInput
				if ui == nil {
					continue
				}
				if !pgate.deliver(ui.GetPromptId(), ui.GetReply()) {
					_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
						Message: "no matching permission prompt for prompt_id",
						Code:    "permission_mismatch",
					}}})
				}

			case *openclaudev4.ClientMessage_Cancel:
				cancel()
			default:
				_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
					Message: "empty client message",
					Code:    "invalid_argument",
				}}})
			}
		}
	}
}

type permGate struct {
	mu sync.Mutex
	id string
	ch chan bool
}

func (g *permGate) wait(ctx context.Context, send func(*openclaudev4.ServerMessage) error, toolName string, args map[string]any) bool {
	id := newPromptID()
	argsJSON := ""
	if args != nil {
		b, err := json.Marshal(args)
		if err == nil {
			argsJSON = string(b)
		}
	}
	ch := make(chan bool, 1)

	g.mu.Lock()
	g.id = id
	g.ch = ch
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		if g.id == id {
			g.id = ""
			g.ch = nil
		}
		g.mu.Unlock()
	}()

	q := fmt.Sprintf("Approve tool %q with args %s?", toolName, core.FormatToolArgsForLog(args))
	_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_PermissionRequired{PermissionRequired: &openclaudev4.PermissionRequired{
		PromptId:       id,
		ToolName:       toolName,
		ArgumentsJson:  argsJSON,
		Question:       q,
	}}})

	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		return false
	}
}

func (g *permGate) deliver(promptID, reply string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.id == "" || g.ch == nil || promptID != g.id {
		return false
	}
	ch := g.ch
	g.ch = nil
	g.id = ""
	ch <- parseApproval(reply)
	return true
}

func parseApproval(reply string) bool {
	s := strings.TrimSpace(strings.ToLower(reply))
	return s == "y" || s == "yes"
}

func newPromptID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "perm-fallback"
	}
	return hex.EncodeToString(b[:])
}

func serverMessageFromEvent(e core.Event) *openclaudev4.ServerMessage {
	switch e.Kind {
	case core.KindUserMessage:
		return nil
	case core.KindAssistantTextDelta:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_TextChunk{TextChunk: &openclaudev4.TextChunk{
			Text: e.TextChunk,
		}}}
	case core.KindAssistantFinished:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_AssistantFinished{AssistantFinished: &openclaudev4.AssistantFinished{
			FullText:      e.AssistantText,
			ToolCallCount: int32(e.ToolCallCount),
			FinishReason:  e.FinishReason,
			AssistantRound: int32(e.AssistantRounds),
		}}}
	case core.KindModelRefusal:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
			Message: e.Message,
			Code:    "model_refusal",
		}}}
	case core.KindToolCall:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_ToolStart{ToolStart: &openclaudev4.ToolCallStart{
			ToolName:      e.ToolName,
			ArgumentsJson: e.ToolArgsJSON,
			ToolUseId:     e.ToolCallID,
		}}}
	case core.KindPermissionPrompt:
		// PermissionRequired is sent from Confirm with a prompt id; omit duplicate event.
		return nil
	case core.KindPermissionResult:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_PermissionAck{PermissionAck: &openclaudev4.PermissionAck{
			PromptId:  e.PermissionTool,
			Approved:  e.PermissionApproved,
		}}}
	case core.KindToolResult:
		isErr := e.ToolExecError != ""
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_ToolResult{ToolResult: &openclaudev4.ToolCallResult{
			ToolName:     e.ToolName,
			ToolUseId:    e.ToolCallID,
			Output:       e.ToolResultText,
			IsError:      isErr,
			ErrorMessage: e.ToolExecError,
		}}}
	case core.KindError:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
			Message: e.Message,
			Code:    "kernel_error",
		}}}
	case core.KindTurnComplete:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_TurnComplete{TurnComplete: &openclaudev4.TurnComplete{}}}
	default:
		return nil
	}
}
