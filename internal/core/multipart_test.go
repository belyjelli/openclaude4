package core

import (
	"strings"
	"testing"

	sdk "github.com/sashabaranov/go-openai"
)

func TestBuildUserContentPartsFromGRPC_URLOnly(t *testing.T) {
	parts, err := BuildUserContentPartsFromGRPC("", []string{"https://example.com/a.png"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) < 2 {
		t.Fatalf("want text + image parts, got %d", len(parts))
	}
	if parts[0].Type != sdk.ChatMessagePartTypeText {
		t.Fatalf("first part type = %q", parts[0].Type)
	}
	if parts[1].Type != sdk.ChatMessagePartTypeImageURL || parts[1].ImageURL == nil || !strings.Contains(parts[1].ImageURL.URL, "example.com") {
		t.Fatalf("bad image part: %+v", parts[1])
	}
}

func TestBuildUserContentPartsFromGRPC_tooMany(t *testing.T) {
	urls := make([]string, MaxGRPCImageAttachments+1)
	for i := range urls {
		urls[i] = "https://example.com/x.png"
	}
	_, err := BuildUserContentPartsFromGRPC("hi", urls, nil)
	if err == nil {
		t.Fatal("expected error for too many images")
	}
}
