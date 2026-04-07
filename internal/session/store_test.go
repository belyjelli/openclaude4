package session

import (
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHandle(dir, "t1")
	if err != nil {
		t.Fatal(err)
	}
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: "sys"},
		{Role: sdk.ChatMessageRoleUser, Content: "hi"},
	}
	if err := h.SaveFrom(msgs, "/tmp/wd"); err != nil {
		t.Fatal(err)
	}
	var loaded []sdk.ChatCompletionMessage
	if err := h.LoadInto(&loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[1].Content != "hi" {
		t.Fatalf("loaded %+v", loaded)
	}
}

func TestListAndLatest(t *testing.T) {
	dir := t.TempDir()
	a, _ := NewHandle(dir, "a")
	b, _ := NewHandle(dir, "b")
	_ = a.SaveFrom([]sdk.ChatCompletionMessage{{Role: sdk.ChatMessageRoleUser, Content: "1"}}, "")
	_ = b.SaveFrom([]sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "2"},
		{Role: sdk.ChatMessageRoleUser, Content: "3"},
	}, "")
	list, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 entries, got %d", len(list))
	}
	latest, err := LatestName(dir)
	if err != nil || latest != "b" {
		t.Fatalf("LatestName = %q %v", latest, err)
	}
	if list[0].Name != "b" {
		t.Fatalf("want newest first by Updated, got %v", list[0].Name)
	}
	if list[0].NMsgs != 2 {
		t.Fatalf("NMsgs %d", list[0].NMsgs)
	}
}

func TestDefaultDir_suffix(t *testing.T) {
	d, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("openclaude", "sessions")
	if !strings.HasSuffix(filepath.Clean(d), want) {
		t.Fatalf("got %s", d)
	}
}

func TestRepairInterruptedToolRound(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: "s"},
		{Role: sdk.ChatMessageRoleUser, Content: "run tools"},
		{
			Role: sdk.ChatMessageRoleAssistant,
			ToolCalls: []sdk.ToolCall{
				{
					ID:   "c1",
					Type: sdk.ToolTypeFunction,
					Function: sdk.FunctionCall{
						Name:      "FileRead",
						Arguments: `{"file_path":"x"}`,
					},
				},
				{
					ID:   "c2",
					Type: sdk.ToolTypeFunction,
					Function: sdk.FunctionCall{
						Name:      "Bash",
						Arguments: `{"command":"echo hi"}`,
					},
				},
			},
		},
		{
			Role:       sdk.ChatMessageRoleTool,
			ToolCallID: "c1",
			Name:       "FileRead",
			Content:    "ok",
		},
	}
	fixed := RepairTranscript(msgs)
	if len(fixed) != 5 {
		t.Fatalf("len=%d want 5", len(fixed))
	}
	last := fixed[len(fixed)-1]
	if last.Role != sdk.ChatMessageRoleTool || last.ToolCallID != "c2" {
		t.Fatalf("last msg = %#v", last)
	}
	if !strings.Contains(last.Content, "recovery") {
		t.Fatalf("content %q", last.Content)
	}
}
