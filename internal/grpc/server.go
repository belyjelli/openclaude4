// Package ocrpc implements the OpenClaude v4 gRPC API over the shared kernel ([core.Agent]).
package ocrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/grpc/openclaudev4"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func grpcNonEmptyImageURLs(urls []string) bool {
	for _, u := range urls {
		if strings.TrimSpace(u) != "" {
			return true
		}
	}
	return false
}

func grpcInlineImagesFromProto(attachments []*openclaudev4.ImageAttachment) []core.GRPCInlineImage {
	var out []core.GRPCInlineImage
	for _, a := range attachments {
		if a == nil || len(a.GetData()) == 0 {
			continue
		}
		out = append(out, core.GRPCInlineImage{
			Data: a.GetData(),
			MIME: a.GetMimeType(),
		})
	}
	return out
}

// Kernel wires the same [core.StreamClient] and [tools.Registry] types as the CLI/TUI.
type Kernel struct {
	Client      core.StreamClient
	Registry    *tools.Registry
	AutoApprove bool
	// TaskParent, when non-nil, is set to the active [core.Agent] for each RunUserTurn so the Task tool resolves the correct parent (serialized by [AgentService.serveTurnMu]).
	TaskParent *atomic.Pointer[core.Agent]
	// Session enables on-disk transcript load/save per stream when Disabled is false and Dir is non-empty.
	Session SessionOpts
}

// SessionOpts configures optional gRPC session_id binding to [session.Store] files.
type SessionOpts struct {
	Disabled bool
	Dir      string
}

// AgentService implements [openclaudev4.AgentServiceServer].
type AgentService struct {
	openclaudev4.UnimplementedAgentServiceServer
	Kernel      Kernel
	serveTurnMu sync.Mutex // one RunUserTurn at a time across streams (Task tool + session consistency)
}

// Register mounts AgentService on the given gRPC registrar (typically [*grpc.Server]).
func Register(s grpc.ServiceRegistrar, k Kernel) {
	openclaudev4.RegisterAgentServiceServer(s, &AgentService{Kernel: k})
}

func bindChatRequestSession(req *openclaudev4.ChatRequest, dir string, streamID *string, store **session.Store, messages *[]sdk.ChatCompletionMessage, wdSave string) error {
	reqSID := strings.TrimSpace(req.GetSessionId())
	target := reqSID
	if target == "" {
		if *streamID == "" {
			*streamID = session.NewRandomID()
		}
		target = *streamID
	} else {
		*streamID = target
	}

	if *store != nil && (*store).ID == target {
		return nil
	}

	if *store != nil {
		_ = (*store).Save(session.RepairTranscript(*messages), wdSave)
	}

	*store = &session.Store{Dir: dir, ID: target}
	data, err := (*store).Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		*messages = nil
		return fmt.Errorf("session load %q: %w", target, err)
	}
	if err == nil {
		*messages = session.RepairTranscript(data.Messages)
	} else {
		*messages = nil
	}
	return nil
}

// Chat implements the bidirectional chat stream.
func (s *AgentService) Chat(stream grpc.BidiStreamingServer[openclaudev4.ClientMessage, openclaudev4.ServerMessage]) error {
	if s.Kernel.Client == nil || s.Kernel.Registry == nil {
		return status.Error(codes.FailedPrecondition, "kernel: missing client or registry")
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	clientCh := make(chan *openclaudev4.ClientMessage, 16)
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
	var streamSessionID string
	var activeStore *session.Store
	var turnWG sync.WaitGroup
	pgate := new(permGate)

	// Correlate PermissionAck with PermissionRequired.prompt_id (kernel events use tool name only).
	tl := new(turnLocal)

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
				if req == nil {
					_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
						Message: "chat_request is required",
						Code:    "invalid_argument",
					}}})
					continue
				}
				userText := strings.TrimSpace(req.GetUserText())
				inlines := grpcInlineImagesFromProto(req.GetImageInline())
				hasImg := grpcNonEmptyImageURLs(req.GetImageUrl()) || len(inlines) > 0
				if userText == "" && !hasImg {
					_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
						Message: "chat_request.user_text or at least one image (image_url / image_inline) is required",
						Code:    "invalid_argument",
					}}})
					continue
				}
				turnWG.Add(1)
				urlCopy := append([]string(nil), req.GetImageUrl()...)
				inlineCopy := append([]core.GRPCInlineImage(nil), inlines...)
				go func(req *openclaudev4.ChatRequest, userText string, imageURLs []string, inlines []core.GRPCInlineImage, hasImg bool) {
					defer turnWG.Done()
					s.serveTurnMu.Lock()
					defer s.serveTurnMu.Unlock()

					turnCtx := ctx
					wdRel := strings.TrimSpace(req.GetWorkingDirectory())
					if wdRel != "" {
						turnCtx = tools.WithWorkDir(turnCtx, wdRel)
					}
					wdSave := wdRel
					if wdSave == "" {
						wdSave = "."
					}
					if awd, err := filepath.Abs(wdSave); err == nil {
						wdSave = awd
					}

					sess := s.Kernel.Session
					persist := !sess.Disabled && strings.TrimSpace(sess.Dir) != ""
					if persist {
						if err := bindChatRequestSession(req, sess.Dir, &streamSessionID, &activeStore, &messages, wdSave); err != nil {
							_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
								Message: err.Error(),
								Code:    "session",
							}}})
							return
						}
					}

					_ = session.ApplyTokenThreshold(turnCtx, s.Kernel.Client, &messages,
						config.SessionCompactTokenThreshold(),
						config.SessionSummarizeOverThreshold(),
						config.SessionCompactKeepMessages(),
						core.EffectiveSystemPrompt(),
					)

					agent := &core.Agent{
						Client:   s.Kernel.Client,
						Registry: s.Kernel.Registry,
						Out:      io.Discard,
						OnEvent: func(e core.Event) {
							if msg := serverMessageFromEvent(e, tl); msg != nil {
								_ = send(msg)
							}
						},
						Confirm: func(toolName string, args map[string]any) bool {
							if s.Kernel.AutoApprove {
								return true
							}
							return pgate.wait(ctx, send, tl, toolName, args)
						},
					}
					if s.Kernel.TaskParent != nil {
						s.Kernel.TaskParent.Store(agent)
						defer s.Kernel.TaskParent.Store(nil)
					}
					if hasImg {
						parts, err := core.BuildUserContentPartsFromGRPC(req.GetUserText(), imageURLs, inlines)
						if err != nil {
							_ = send(&openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_Error{Error: &openclaudev4.ErrorEvent{
								Message: err.Error(),
								Code:    "invalid_argument",
							}}})
							return
						}
						_ = agent.RunUserTurnMulti(turnCtx, &messages, parts)
					} else {
						_ = agent.RunUserTurn(turnCtx, &messages, userText)
					}

					if persist && activeStore != nil {
						_ = activeStore.Save(session.RepairTranscript(messages), wdSave)
					}
				}(req, userText, urlCopy, inlineCopy, hasImg)

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

// turnLocal holds wire-only correlation for permission messages (not in core.Event).
type turnLocal struct {
	mu                 sync.Mutex
	activePermPromptID string
}

func (t *turnLocal) setPermPromptID(id string) {
	t.mu.Lock()
	t.activePermPromptID = id
	t.mu.Unlock()
}

func (t *turnLocal) takePermPromptID() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := t.activePermPromptID
	t.activePermPromptID = ""
	return id
}

type permGate struct {
	mu sync.Mutex
	id string
	ch chan bool
}

func (g *permGate) wait(ctx context.Context, send func(*openclaudev4.ServerMessage) error, tl *turnLocal, toolName string, args map[string]any) bool {
	id := newPromptID()
	tl.setPermPromptID(id)
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
		PromptId:      id,
		ToolName:      toolName,
		ArgumentsJson: argsJSON,
		Question:      q,
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

func serverMessageFromEvent(e core.Event, tl *turnLocal) *openclaudev4.ServerMessage {
	switch e.Kind {
	case core.KindUserMessage:
		return nil
	case core.KindAssistantTextDelta:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_TextChunk{TextChunk: &openclaudev4.TextChunk{
			Text: e.TextChunk,
		}}}
	case core.KindAssistantFinished:
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_AssistantFinished{AssistantFinished: &openclaudev4.AssistantFinished{
			FullText:       e.AssistantText,
			ToolCallCount:  int32(e.ToolCallCount),
			FinishReason:   e.FinishReason,
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
		pid := ""
		if tl != nil {
			pid = tl.takePermPromptID()
		}
		return &openclaudev4.ServerMessage{Event: &openclaudev4.ServerMessage_PermissionAck{PermissionAck: &openclaudev4.PermissionAck{
			PromptId: pid,
			Approved: e.PermissionApproved,
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
