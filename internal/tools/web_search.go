package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearch uses DuckDuckGo's instant-answer JSON API (no API key; limited results).
type WebSearch struct{}

func (WebSearch) Name() string      { return "WebSearch" }
func (WebSearch) IsDangerous() bool { return false }
func (WebSearch) Description() string {
	return "Lightweight web lookup via DuckDuckGo instant answers (abstract + related topics, not full SERP)."
}

func (WebSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
		"required": []string{"query"},
	}
}

func (WebSearch) Execute(ctx context.Context, args map[string]any) (string, error) {
	q, _ := args["query"].(string)
	if strings.TrimSpace(q) == "" {
		return "", fmt.Errorf("query is required")
	}

	u, err := url.Parse("https://api.duckduckgo.com/")
	if err != nil {
		return "", err
	}
	qv := u.Query()
	qv.Set("q", q)
	qv.Set("format", "json")
	qv.Set("no_html", "1")
	qv.Set("skip_disambig", "1")
	u.RawQuery = qv.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "openclaude4/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Abstract       string `json:"Abstract"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Heading        string `json:"Heading"`
		RelatedTopics  []any  `json:"RelatedTopics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("json: %w", err)
	}

	var b strings.Builder
	if payload.Heading != "" {
		fmt.Fprintf(&b, "## %s\n\n", payload.Heading)
	}
	if payload.Abstract != "" {
		fmt.Fprintf(&b, "%s\n", payload.Abstract)
		if payload.AbstractURL != "" {
			fmt.Fprintf(&b, "\nSource: %s (%s)\n", payload.AbstractURL, payload.AbstractSource)
		}
	}
	// Flatten a few related links (structure varies).
	for _, rt := range payload.RelatedTopics {
		if b.Len() > 8000 {
			break
		}
		if m, ok := rt.(map[string]any); ok {
			if text, ok := m["Text"].(string); ok {
				fmt.Fprintf(&b, "- %s\n", text)
			}
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(no instant answer; try a more specific query)", nil
	}
	return out, nil
}
