package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// ParseGitHubURL returns owner, repo, ref, subPath (directory within repo), ok.
// ref may be empty when the URL uses default branch shorthand (caller should resolve).
func ParseGitHubURL(raw string) (owner, repo, ref, subPath string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", "", false
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", "", "", false
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	if host != "github.com" {
		return "", "", "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", "", "", false
	}
	owner, repo = parts[0], parts[1]
	rest := parts[2:]
	if len(rest) == 0 {
		return owner, repo, "", "", true
	}
	switch strings.ToLower(rest[0]) {
	case "tree", "blob":
		if len(rest) < 2 {
			return owner, repo, "", "", true
		}
		ref = rest[1]
		if len(rest) > 2 {
			subPath = strings.Join(rest[2:], "/")
		}
		return owner, repo, ref, subPath, true
	case "commit":
		if len(rest) >= 2 {
			ref = rest[1]
		}
		return owner, repo, ref, "", true
	default:
		// e.g. /issues — still a valid repo URL, no ref in path
		return owner, repo, "", "", true
	}
}

// RepoAPIMetadata fetches default branch and metadata from GitHub API (unauthenticated ok).
func RepoAPIMetadata(ctx context.Context, client *http.Client, owner, repo string) (defaultBranch, description, language string, topics []string, err error) {
	if client == nil {
		client = http.DefaultClient
	}
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return "", "", "", nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return "", "", "", nil, fmt.Errorf("github api: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var wrap struct {
		DefaultBranch string   `json:"default_branch"`
		Description   string   `json:"description"`
		Language      string   `json:"language"`
		Topics        []string `json:"topics"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return "", "", "", nil, err
	}
	return wrap.DefaultBranch, wrap.Description, wrap.Language, wrap.Topics, nil
}

// FetchRaw downloads a file from raw.githubusercontent.com (ref must be non-empty branch or tag name).
func FetchRaw(ctx context.Context, client *http.Client, owner, repo, ref, repoPath string) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	repoPath = strings.Trim(strings.TrimPrefix(repoPath, "/"), "/")
	p := path.Join("/", owner, repo, ref, repoPath)
	u := "https://raw.githubusercontent.com" + p
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("raw %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// NewHTTPClient returns a client suitable for installer fetches.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
