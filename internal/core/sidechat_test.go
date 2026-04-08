package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestSideQuestion(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(sseBody(sdk.ChatCompletionStreamResponse{
			Choices: []sdk.ChatCompletionStreamChoice{{
				Delta: sdk.ChatCompletionStreamChoiceDelta{Content: "short answer"},
			}},
		}))
	}))
	defer srv.Close()
	c := newTestStreamClient(t, srv)
	ans, err := SideQuestion(context.Background(), c, "what is 2+2?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ans, "short answer") {
		t.Fatalf("got %q", ans)
	}
}

func TestSideQuestion_Empty(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	c := newTestStreamClient(t, srv)
	_, err := SideQuestion(context.Background(), c, "  ")
	if err == nil {
		t.Fatal("expected error")
	}
}
