package providerwizard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ListOllamaModelTags GETs Ollama /api/tags and returns model names, or error on failure.
func ListOllamaModelTags(host string) ([]string, error) {
	h := strings.TrimSpace(host)
	if h == "" {
		h = "http://127.0.0.1:11434"
	}
	h = strings.TrimRight(h, "/")
	if strings.HasSuffix(h, "/v1") {
		h = strings.TrimSuffix(h, "/v1")
	}
	u := h + "/api/tags"
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	var doc struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.Models))
	for _, m := range doc.Models {
		n := strings.TrimSpace(m.Name)
		if n != "" {
			out = append(out, n)
		}
	}
	return out, nil
}
