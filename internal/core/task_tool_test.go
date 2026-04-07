package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

func TestTaskTool_Execute_MaxDepth(t *testing.T) {
	var agent *Agent
	reg := tools.NewRegistry()
	reg.Register(NewTaskTool(func() *Agent { return agent }))
	agent = &Agent{
		Client:   newStubFailClient(t),
		Registry: reg,
		Out:      io.Discard,
	}
	tt, _ := reg.Get("Task")
	ctx := tools.WithSubTaskDepth(context.Background(), 1)
	_, err := tt.Execute(ctx, map[string]any{"goal": "x"})
	if err == nil || !strings.Contains(err.Error(), "max nested depth") {
		t.Fatalf("expected depth error, got %v", err)
	}
}

func TestTaskTool_Execute_SubAgentReplies(t *testing.T) {
	body := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "done"}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	var agent *Agent
	reg := tools.NewRegistry()
	reg.Register(NewTaskTool(func() *Agent { return agent }))
	agent = &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: reg,
		Out:      io.Discard,
	}
	tt, ok := reg.Get("Task")
	if !ok {
		t.Fatal("missing Task")
	}
	out, err := tt.Execute(context.Background(), map[string]any{"goal": "summarize"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("got %q", out)
	}
}

// newStubFailClient fails if StreamChatWithTools is called (for depth test).
type stubFailClient struct {
	t *testing.T
}

func newStubFailClient(t *testing.T) *stubFailClient {
	return &stubFailClient{t: t}
}

func (c *stubFailClient) StreamChatWithTools(ctx context.Context, messages []sdk.ChatCompletionMessage, toolList []sdk.Tool) (*sdk.ChatCompletionStream, error) {
	c.t.Fatal("unexpected API call")
	return nil, nil
}
