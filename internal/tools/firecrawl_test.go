package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFirecrawlEnabled_unset(t *testing.T) {
	t.Setenv("FIRECRAWL_API_KEY", "")
	if FirecrawlEnabled() {
		t.Fatal("expected disabled")
	}
}

func TestFirecrawlEnabled_set(t *testing.T) {
	t.Setenv("FIRECRAWL_API_KEY", "abc")
	if !FirecrawlEnabled() {
		t.Fatal("expected enabled")
	}
}

func TestFirecrawlScrapeMarkdown_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scrape" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"markdown": "# Title\n\nbody"},
		})
	}))
	defer srv.Close()
	t.Cleanup(func() { firecrawlTestBaseURL = "" })
	firecrawlTestBaseURL = srv.URL

	md, err := firecrawlScrapeMarkdown(context.Background(), "test-key", "https://example.com/page")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Title") {
		t.Fatalf("got %q", md)
	}
}

func TestFirecrawlSearchResults_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]any{
				{"title": "A", "description": "desc", "url": "https://a.test"},
			},
		})
	}))
	defer srv.Close()
	t.Cleanup(func() { firecrawlTestBaseURL = "" })
	firecrawlTestBaseURL = srv.URL

	out, err := firecrawlSearchResults(context.Background(), "k", "q")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Firecrawl") || !strings.Contains(out, "https://a.test") {
		t.Fatalf("got %q", out)
	}
}

func TestTruncateRunesWeb(t *testing.T) {
	t.Parallel()
	s := strings.Repeat("é", 5) // 5 runes, 10 bytes
	got, note := truncateRunesWeb(s, 3)
	if utf8Len := len([]rune(got)); utf8Len != 3 {
		t.Fatalf("len %d", utf8Len)
	}
	if note == "" {
		t.Fatal("want note")
	}
}
