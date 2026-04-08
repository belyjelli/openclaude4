package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/session"
	sdk "github.com/sashabaranov/go-openai"
)

func TestSlashExportEmpty(t *testing.T) {
	var empty []sdk.ChatCompletionMessage
	st := chatState{messages: &empty}
	var buf bytes.Buffer
	if err := slashExport(st, nil, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no messages") {
		t.Fatalf("expected empty hint, got %q", buf.String())
	}
}

func TestSlashExportJSONRoundTrip(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: "sys"},
		{Role: sdk.ChatMessageRoleUser, Content: "hi"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "hello"},
	}
	st := chatState{messages: &msgs}
	var buf bytes.Buffer
	if err := slashExport(st, []string{"json"}, &buf); err != nil {
		t.Fatal(err)
	}
	var got session.FileV1
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\n%s", err, buf.String())
	}
	if got.Version != 1 {
		t.Fatalf("version: %d", got.Version)
	}
	if got.ID != "inline-export" {
		t.Fatalf("id: %q", got.ID)
	}
	if len(got.Messages) != 3 {
		t.Fatalf("messages len: %d", len(got.Messages))
	}
}

func TestSlashExportMarkdown(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "q"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "a"},
	}
	st := chatState{messages: &msgs}
	var buf bytes.Buffer
	if err := slashExport(st, []string{"md"}, &buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "## user") || !strings.Contains(s, "## assistant") {
		t.Fatalf("unexpected md: %s", s)
	}
}

func TestSlashExportJSONToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "x"},
	}
	st := chatState{messages: &msgs}
	var buf bytes.Buffer
	if err := slashExport(st, []string{path}, &buf); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got session.FileV1
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("msgs: %d", len(got.Messages))
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o777 != 0o600 {
		t.Fatalf("want 0600, got %o", fi.Mode().Perm()&0o777)
	}
}

func TestSlashExportOversizeInlineJSON(t *testing.T) {
	huge := strings.Repeat("x", maxInlineExportBytes+1)
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: huge},
	}
	st := chatState{messages: &msgs}
	var buf bytes.Buffer
	err := slashExport(st, nil, &buf)
	if err == nil {
		t.Fatal("expected error for oversized inline JSON")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Fatalf("expected path hint in error: %v", err)
	}
}

func TestSlashExportRejectDir(t *testing.T) {
	dir := t.TempDir()
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "x"},
	}
	st := chatState{messages: &msgs}
	err := slashExport(st, []string{dir}, io.Discard)
	if err == nil {
		t.Fatal("expected error for directory path")
	}
}
