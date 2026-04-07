package ocrpc

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlawb/openclaude4/internal/grpc/openclaudev4"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20

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

func TestChat_streamingText(t *testing.T) {
	body := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "hi"}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	cfg := sdk.DefaultConfig("sk-test")
	cfg.BaseURL = strings.TrimSuffix(srv.URL, "/") + "/v1"
	client := &streamSDKClient{inner: sdk.NewClientWithConfig(cfg), model: sdk.GPT4oMini}

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })

	gs := grpc.NewServer()
	Register(gs, Kernel{Client: client, Registry: tools.NewDefaultRegistry(), AutoApprove: true})
	go func() {
		if err := gs.Serve(lis); err != nil {
			t.Logf("grpc serve: %v", err)
		}
	}()
	t.Cleanup(gs.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	grpcClient := openclaudev4.NewAgentServiceClient(conn)
	stream, err := grpcClient.Chat(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := stream.Send(&openclaudev4.ClientMessage{Payload: &openclaudev4.ClientMessage_ChatRequest{
		ChatRequest: &openclaudev4.ChatRequest{UserText: "hello"},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}

	var sawText, sawFinished, sawTurn bool
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch ev := msg.GetEvent().(type) {
		case *openclaudev4.ServerMessage_TextChunk:
			if ev.TextChunk.GetText() == "hi" {
				sawText = true
			}
		case *openclaudev4.ServerMessage_AssistantFinished:
			if ev.AssistantFinished.GetFullText() == "hi" {
				sawFinished = true
			}
		case *openclaudev4.ServerMessage_TurnComplete:
			sawTurn = true
		case *openclaudev4.ServerMessage_Error:
			t.Fatalf("unexpected error: %s", ev.Error.GetMessage())
		}
	}

	if !sawText {
		t.Error("expected text chunk hi")
	}
	if !sawFinished {
		t.Error("expected assistant_finished")
	}
	if !sawTurn {
		t.Error("expected turn_complete")
	}
}

func TestChat_persistsSessionWhenConfigured(t *testing.T) {
	body := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "ok"}},
			},
		},
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{}, FinishReason: sdk.FinishReasonStop},
			},
		},
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	cfg := sdk.DefaultConfig("sk-test")
	cfg.BaseURL = strings.TrimSuffix(srv.URL, "/") + "/v1"
	client := &streamSDKClient{inner: sdk.NewClientWithConfig(cfg), model: sdk.GPT4oMini}

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })

	dir := t.TempDir()
	sid := "grpctest"
	gs := grpc.NewServer()
	Register(gs, Kernel{
		Client:      client,
		Registry:    tools.NewDefaultRegistry(),
		AutoApprove: true,
		Session:     SessionOpts{Disabled: false, Dir: dir},
	})
	go func() {
		if err := gs.Serve(lis); err != nil {
			t.Logf("grpc serve: %v", err)
		}
	}()
	t.Cleanup(gs.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	grpcClient := openclaudev4.NewAgentServiceClient(conn)
	stream, err := grpcClient.Chat(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := stream.Send(&openclaudev4.ClientMessage{Payload: &openclaudev4.ClientMessage_ChatRequest{
		ChatRequest: &openclaudev4.ChatRequest{UserText: "hello", SessionId: &sid},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	st := &session.Store{Dir: dir, ID: sid}
	data, err := st.Load()
	if err != nil {
		t.Fatalf("session file: %v (want %s)", err, filepath.Join(dir, sid+".json"))
	}
	if len(data.Messages) == 0 {
		t.Fatal("expected persisted transcript")
	}
}
