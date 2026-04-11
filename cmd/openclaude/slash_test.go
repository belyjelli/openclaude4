package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/session"
	"github.com/gitlawb/openclaude4/internal/skills"
	sdk "github.com/sashabaranov/go-openai"
)

func TestCompactTail_SystemAndRest(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleSystem, Content: "sys"},
		{Role: sdk.ChatMessageRoleUser, Content: "u1"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "a1"},
		{Role: sdk.ChatMessageRoleUser, Content: "u2"},
		{Role: sdk.ChatMessageRoleAssistant, Content: "a2"},
		{Role: sdk.ChatMessageRoleUser, Content: "u3"},
	}
	out := session.CompactTail(msgs, 2)
	if len(out) != 3 {
		t.Fatalf("len %d want 3: %+v", len(out), out)
	}
	if out[0].Content != "sys" || out[1].Content != "a2" || out[2].Content != "u3" {
		t.Fatalf("wrong tail: %+v", out)
	}
}

func TestCompactTail_NoSystem(t *testing.T) {
	msgs := []sdk.ChatCompletionMessage{
		{Role: sdk.ChatMessageRoleUser, Content: "a"},
		{Role: sdk.ChatMessageRoleUser, Content: "b"},
		{Role: sdk.ChatMessageRoleUser, Content: "c"},
	}
	out := session.CompactTail(msgs, 2)
	if len(out) != 2 || out[0].Content != "b" || out[1].Content != "c" {
		t.Fatalf("got %+v", out)
	}
}

func TestSkillSlashInlineSlashSubmitUser(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "demo")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(sub, "SKILL.md")
	body := "---\nname: demo\ndescription: x\n---\nSay $ARGUMENTS"
	if err := os.WriteFile(md, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := skills.Load([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	st := chatState{
		skillCat:     cat,
		ctx:          context.Background(),
		resolveAgent: func() *core.Agent { return nil },
	}
	err = handleSlashLine("/demo hello # tail", st, nil)
	var su core.SlashSubmitUser
	if !errors.As(err, &su) {
		t.Fatalf("want SlashSubmitUser, got %v", err)
	}
	if su.UserText == "" || !strings.Contains(su.UserText, "hello") {
		t.Fatalf("unexpected user text: %q", su.UserText)
	}
}
