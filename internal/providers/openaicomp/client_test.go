package openaicomp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/config"
	sdk "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func TestStreamChatWithTools_SendsToolsInBody(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		captured = b
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	viper.Reset()
	config.Load("")
	viper.Set("openai.api_key", "sk-test")
	viper.Set("provider.base_url", strings.TrimSuffix(srv.URL, "/")+"/v1")

	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	tool := sdk.Tool{
		Type: sdk.ToolTypeFunction,
		Function: &sdk.FunctionDefinition{
			Name:        "noop",
			Description: "test",
		},
	}
	stream, err := c.StreamChatWithTools(context.Background(), []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "hi"},
	}, []sdk.Tool{tool})
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err := stream.Recv()
		if err != nil {
			break
		}
	}
	_ = stream.Close()

	var body map[string]json.RawMessage
	if err := json.Unmarshal(captured, &body); err != nil {
		t.Fatalf("request body json: %v", err)
	}
	if _, ok := body["tools"]; !ok {
		t.Fatalf("expected tools in request body, got keys: %v", bodyKeys(body))
	}
}

func bodyKeys(m map[string]json.RawMessage) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	return k
}
