package core

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
	sdk "github.com/sashabaranov/go-openai"
)

func TestEventHarness_TextOnly(t *testing.T) {
	body := sseBody(
		sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{
				{Index: 0, Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "Hi"}},
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
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	var evs []Event
	var out bytes.Buffer
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: tools.NewRegistry(),
		Out:      &out,
		OnEvent:  func(e Event) { evs = append(evs, e) },
	}
	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(context.Background(), &messages, "yo"); err != nil {
		t.Fatal(err)
	}

	if len(evs) < 4 {
		t.Fatalf("expected at least 4 events, got %d: %#v", len(evs), evs)
	}
	if evs[0].Kind != KindUserMessage || evs[0].UserText != "yo" {
		t.Fatalf("first event = %#v", evs[0])
	}
	if evs[1].Kind != KindAssistantTextDelta || evs[1].TextChunk != "Hi" {
		t.Fatalf("delta event = %#v", evs[1])
	}
	if evs[2].Kind != KindAssistantFinished || evs[2].AssistantText != "Hi" || evs[2].ToolCallCount != 0 {
		t.Fatalf("finished event = %#v", evs[2])
	}
	if evs[len(evs)-1].Kind != KindTurnComplete {
		t.Fatalf("last event = %#v", evs[len(evs)-1])
	}
}

func TestEventHarness_ToolThenText(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("inside"), 0o644); err != nil {
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

	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		switch atomicAddInt(&n, 1) {
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

	var evs []Event
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: reg,
		Out:      &bytes.Buffer{},
		OnEvent:  func(e Event) { evs = append(evs, e) },
	}
	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(ctx, &messages, "read"); err != nil {
		t.Fatal(err)
	}

	var kinds []EventKind
	for _, e := range evs {
		kinds = append(kinds, e.Kind)
	}
	if !containsSeq(kinds, KindUserMessage, KindAssistantFinished, KindToolCall, KindToolResult, KindAssistantFinished, KindTurnComplete) {
		t.Fatalf("event sequence: %v", kinds)
	}
}

func intPtr(i int) *int { return &i }

func atomicAddInt(p *int32, d int32) int32 {
	*p += d
	return *p
}

func containsSeq(haystack []EventKind, needle ...EventKind) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		ok := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func TestEventHarness_StreamError(t *testing.T) {
	bad := []byte("data: {not-json}\n\ndata: [DONE]\n\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(bad)
	}))
	t.Cleanup(srv.Close)

	var evs []Event
	agent := &Agent{
		Client:   newTestStreamClient(t, srv),
		Registry: tools.NewRegistry(),
		Out:      &bytes.Buffer{},
		OnEvent:  func(e Event) { evs = append(evs, e) },
	}
	var messages []sdk.ChatCompletionMessage
	if err := agent.RunUserTurn(context.Background(), &messages, "x"); err == nil {
		t.Fatal("expected error")
	}
	s := ""
	for _, e := range evs {
		s += string(e.Kind) + ","
	}
	if !strings.Contains(s, string(KindUserMessage)) {
		t.Fatalf("missing user event: %s", s)
	}
	if !strings.Contains(s, string(KindError)) {
		t.Fatalf("missing error event: %s", s)
	}
}
