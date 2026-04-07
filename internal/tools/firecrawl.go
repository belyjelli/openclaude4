package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	firecrawlAPIBaseProd   = "https://api.firecrawl.dev/v1"
	firecrawlScrapeTimeout = 60 * time.Second
	firecrawlSearchTimeout = 45 * time.Second
	firecrawlMaxReadBytes  = 4 << 20 // 4 MiB JSON body cap
	firecrawlSearchLimit   = 10
)

// firecrawlTestBaseURL, when non-empty (tests only), overrides the Firecrawl API base URL.
var firecrawlTestBaseURL string

func firecrawlBaseURL() string {
	if firecrawlTestBaseURL != "" {
		return strings.TrimRight(firecrawlTestBaseURL, "/")
	}
	return firecrawlAPIBaseProd
}

// FirecrawlEnabled is true when FIRECRAWL_API_KEY is set (non-whitespace).
// WebSearch and WebFetch prefer Firecrawl when enabled, then fall back to DDG / direct HTTP.
func FirecrawlEnabled() bool {
	return strings.TrimSpace(os.Getenv("FIRECRAWL_API_KEY")) != ""
}

func firecrawlAPIKey() string {
	return strings.TrimSpace(os.Getenv("FIRECRAWL_API_KEY"))
}

// firecrawlScrapeMarkdown calls POST /v1/scrape (markdown). Empty markdown is an error for callers that want fallback.
func firecrawlScrapeMarkdown(ctx context.Context, apiKey, pageURL string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"url":     pageURL,
		"formats": []string{"markdown"},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, firecrawlBaseURL()+"/scrape", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "openclaude4/1.0 (Firecrawl)")

	client := &http.Client{Timeout: firecrawlScrapeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, firecrawlMaxReadBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > firecrawlMaxReadBytes {
		return "", fmt.Errorf("firecrawl response too large")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firecrawl scrape: HTTP %d", resp.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
		Data    *struct {
			Markdown string `json:"markdown"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("firecrawl scrape json: %w", err)
	}
	if !out.Success {
		if out.Error != "" {
			return "", fmt.Errorf("firecrawl: %s", out.Error)
		}
		return "", fmt.Errorf("firecrawl scrape unsuccessful")
	}
	if out.Data == nil {
		return "", fmt.Errorf("firecrawl: missing data")
	}
	md := strings.TrimSpace(out.Data.Markdown)
	if md == "" {
		return "", fmt.Errorf("firecrawl: empty markdown")
	}
	return md, nil
}

// firecrawlSearchResults calls POST /v1/search (metadata only; no per-result scrape to save credits).
func firecrawlSearchResults(ctx context.Context, apiKey, query string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"query": query,
		"limit": firecrawlSearchLimit,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, firecrawlBaseURL()+"/search", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "openclaude4/1.0 (Firecrawl)")

	client := &http.Client{Timeout: firecrawlSearchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, firecrawlMaxReadBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > firecrawlMaxReadBytes {
		return "", fmt.Errorf("firecrawl response too large")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firecrawl search: HTTP %d", resp.StatusCode)
	}

	var out struct {
		Success bool `json:"success"`
		Data    []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("firecrawl search json: %w", err)
	}
	if !out.Success {
		if out.Error != "" {
			return "", fmt.Errorf("firecrawl: %s", out.Error)
		}
		return "", fmt.Errorf("firecrawl search unsuccessful")
	}
	if len(out.Data) == 0 {
		return "", fmt.Errorf("firecrawl: no search results")
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "## Web search (Firecrawl)\n\n")
	for _, hit := range out.Data {
		title := strings.TrimSpace(hit.Title)
		if title == "" {
			title = hit.URL
		}
		_, _ = fmt.Fprintf(&b, "### %s\n", title)
		if d := strings.TrimSpace(hit.Description); d != "" {
			_, _ = fmt.Fprintf(&b, "%s\n\n", d)
		}
		_, _ = fmt.Fprintf(&b, "URL: %s\n\n", hit.URL)
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return "", fmt.Errorf("firecrawl: empty formatted results")
	}
	return s, nil
}

func truncateRunesWeb(s string, maxChars int) (text, note string) {
	if maxChars <= 0 {
		return s, ""
	}
	if utf8.RuneCountInString(s) <= maxChars {
		return s, ""
	}
	return string([]rune(s)[:maxChars]), fmt.Sprintf("\n\n[truncated to %d characters]", maxChars)
}
