package providers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gitlawb/openclaude4/internal/config"
)

func pingGemini() string {
	key := config.GeminiAPIKey()
	if key == "" {
		return "Gemini: no API key configured"
	}
	u := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", url.QueryEscape(key))
	return pingHTTP(u)
}

func pingHTTP(url string) string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("reachability: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("reachability: OK (%s)", strings.TrimPrefix(url, "http://"))
	}
	return fmt.Sprintf("reachability: HTTP %s", resp.Status)
}
