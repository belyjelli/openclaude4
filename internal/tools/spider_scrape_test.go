package tools

import (
	"context"
	"strings"
	"testing"
)

func TestSpiderScrape_invalidReturnFormat(t *testing.T) {
	_, err := (SpiderScrape{}).Execute(context.Background(), map[string]any{
		"url":           "https://example.com",
		"return_format": "not-a-format",
	})
	if err == nil || !strings.Contains(err.Error(), "return_format") {
		t.Fatalf("want return_format error, got %v", err)
	}
}
