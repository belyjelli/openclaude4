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

func pingOpenRouter() string {
	key := config.OpenRouterAPIKey()
	if key == "" {
		return "OpenRouter: no API key configured"
	}
	u := strings.TrimRight(config.OpenRouterChatBase(), "/") + "/models"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return fmt.Sprintf("OpenRouter: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("OpenRouter reachability: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("OpenRouter reachability: OK (%s)", strings.TrimPrefix(u, "https://"))
	}
	return fmt.Sprintf("OpenRouter reachability: HTTP %s", resp.Status)
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
