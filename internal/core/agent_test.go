package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

// streamSDKClient adapts go-openai's Client to [StreamClient] for tests.
type streamSDKClient struct {
	inner *sdk.Client
	model string
}

func (c *streamSDKClient) StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error) {
	return c.inner.CreateChatCompletionStream(ctx, sdk.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    toolList,
		Stream:   true,
	})
}

func newTestStreamClient(t *testing.T, server *httptest.Server) *streamSDKClient {
	t.Helper()
	cfg := sdk.DefaultConfig("sk-test")
	cfg.BaseURL = strings.TrimSuffix(server.URL, "/") + "/v1"
	return &streamSDKClient{
		inner: sdk.NewClientWithConfig(cfg),
		model: sdk.GPT4oMini,
	}
}

// sseBody builds an OpenAI-style streaming body terminated with [DONE].
func sseBody(chunks ...sdk.ChatCompletionStreamResponse) []byte {
	var b strings.Builder
	for _, ch := range chunks {
		line, err := json.Marshal(ch)
		if err != nil {
			panic(err)
		}
		b.WriteString("data: ")
		b.Write(line)
		b.WriteString("\n\n")
	}
	b.WriteString("data: [DONE]\n\n")
	return []byte(b.String())
}

func ptrIdx(i int) *int { return &i }

func TestRunUserTurn_TextOnly(t *testing.T) {
	body := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "Hello"}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)

	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if n.Add(1) != 1 {
			t.Error("unexpected extra request")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: tools.NewRegistry(),
		Out:      &out,
	}

	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(context.Background(), &messages, "hi"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Hello") {
		t.Fatalf("stdout = %q, want substring Hello", out.String())
	}
	if len(messages) != 3 { // system, user, assistant
		t.Fatalf("len(messages) = %d, want 3", len(messages))
	}
	if messages[2].Role != sdk.ChatMessageRoleAssistant || messages[2].Content != "Hello" {
		t.Fatalf("assistant message = %#v", messages[2])
	}
}

func TestRunUserTurn_ToolThenText(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}

	toolChunk := sdk.ChatCompletionStreamResponse{
		Choices: []sdk.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: sdk.ChatCompletionStreamChoiceDelta{
					ToolCalls: []sdk.ToolCall{
						{
							Index: ptrIdx(0),
							ID:    "call_1",
							Type:  sdk.ToolTypeFunction,
							Function: sdk.FunctionCall{
								Name:      "FileRead",
								Arguments: `{"file_path":"hello.txt"}`,
							},
						},
					},
				},
			},
		},
	}
	finishTools := sdk.ChatCompletionStreamResponse{
		Choices: []sdk.ChatCompletionStreamChoice{
			{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonToolCalls},
		},
	}
	textChunks := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "Done"}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)

	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		switch n.Add(1) {
		case 1:
			_, _ = w.Write(sseBody(toolChunk, finishTools))
		case 2:
			_, _ = w.Write(textChunks)
		default:
			t.Error("unexpected request")
		}
	}))
	t.Cleanup(srv.Close)

	ctx := tools.WithWorkDir(context.Background(), tmp)
	reg := tools.NewRegistry()
	reg.Register(tools.FileRead{})

	var out bytes.Buffer
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: reg,
		Out:      &out,
	}

	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(ctx, &messages, "read the file"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Done") {
		t.Fatalf("stdout = %q, want Done", out.String())
	}
	// system, user, assistant+tool, tool result, assistant final
	if len(messages) != 5 {
		t.Fatalf("len(messages) = %d, want 5", len(messages))
	}
	if messages[3].Role != sdk.ChatMessageRoleTool || !strings.Contains(messages[3].Content, "inside") {
		t.Fatalf("tool message = %#v", messages[3])
	}
}

func TestRunUserTurn_UserDeclinesDangerousTool(t *testing.T) {
	tmp := t.TempDir()
	bashArgs, err := json.Marshal(map[string]string{
		"command": "echo pwn > " + filepath.Join(tmp, "marker"),
	})
	if err != nil {
		t.Fatal(err)
	}

	toolChunk := sdk.ChatCompletionStreamResponse{
		Choices: []sdk.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: sdk.ChatCompletionStreamChoiceDelta{
					ToolCalls: []sdk.ToolCall{
						{
							Index: ptrIdx(0),
							ID:    "call_bash",
							Type:  sdk.ToolTypeFunction,
							Function: sdk.FunctionCall{
								Name:      "Bash",
								Arguments: string(bashArgs),
							},
						},
					},
				},
			},
		},
	}
	finishTools := sdk.ChatCompletionStreamResponse{
		Choices: []sdk.ChatCompletionStreamChoice{
			{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonToolCalls},
		},
	}
	textChunks := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "Understood."}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)

	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		switch n.Add(1) {
		case 1:
			_, _ = w.Write(sseBody(toolChunk, finishTools))
		case 2:
			_, _ = w.Write(textChunks)
		default:
			t.Error("unexpected request")
		}
	}))
	t.Cleanup(srv.Close)

	ctx := tools.WithWorkDir(context.Background(), tmp)
	reg := tools.NewRegistry()
	reg.Register(tools.Bash{})

	var out bytes.Buffer
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: reg,
		Out:      &out,
		Confirm: func(string, map[string]any) bool {
			return false
		},
	}

	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(ctx, &messages, "run a command"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "marker")); !os.IsNotExist(err) {
		t.Fatal("marker should not exist when user declined bash")
	}
	if messages[3].Role != sdk.ChatMessageRoleTool || !strings.Contains(messages[3].Content, "declined") {
		t.Fatalf("tool message = %#v", messages[3])
	}
}

func TestRunUserTurn_MaxIterations(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	toolOnly := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{
					Index: 0,
					Delta: sdk.ChatCompletionStreamChoiceDelta{
						ToolCalls: []sdk.ToolCall{
							{
								Index: ptrIdx(0),
								ID:    "call_x",
								Type:  sdk.ToolTypeFunction,
								Function: sdk.FunctionCall{
									Name:      "FileRead",
									Arguments: `{"file_path":"f.txt"}`,
								},
							},
						},
					},
				},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonToolCalls},
			},
		},
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(toolOnly)
	}))
	t.Cleanup(srv.Close)

	ctx := tools.WithWorkDir(context.Background(), tmp)
	reg := tools.NewRegistry()
	reg.Register(tools.FileRead{})

	agent := &Agent{
		Client:        newTestStreamClient(t, srv),
		Registry:      reg,
		Out:           io.Discard,
		MaxIterations: 2,
	}

	var messages []sdk.ChatCompletionMessage
	err := agent.RunUserTurn(ctx, &messages, "keep reading")
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected iteration limit error, got %v", err)
	}
}
