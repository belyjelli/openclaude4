package session

import (
	"path/filepath"
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestSanitizeName(t *testing.T) {
	if _, err := SanitizeName(""); err == nil {
		t.Fatal("want error for empty")
	}
	if _, err := SanitizeName("../x"); err == nil {
		t.Fatal("want error for path-ish name")
	}
	got, err := SanitizeName("my-work")
	if err != nil || got != "my-work" {
		t.Fatalf("got %q %v", got, err)
	}
}

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
	_ = a.SaveFrom([]sdk.ChatCompletionMessage{{Role: "user", Content: "1"}}, "")
	_ = b.SaveFrom([]sdk.ChatCompletionMessage{{Role: "user", Content: "2"}, {Role: "user", Content: "3"}}, "")
	list, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 entries, got %d", len(list))
	}
	// b should be latest (saved second)
	latest, err := LatestName(dir)
	if err != nil || latest != "b" {
		t.Fatalf("LatestName = %q %v", latest, err)
	}
	if list[0].Name != "b" {
		t.Fatalf("want newest first, got %v", list[0].Name)
	}
	if list[0].NMsgs != 2 {
		t.Fatalf("NMsgs %d", list[0].NMsgs)
	}
}

func TestDefaultDir_env(t *testing.T) {
	t.Setenv("OPENCLAUDE_SESSIONS_DIR", filepath.Join(t.TempDir(), "s"))
	d, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(d) != "s" {
		t.Fatalf("got %s", d)
	}
}
