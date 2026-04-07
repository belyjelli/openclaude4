package providers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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
