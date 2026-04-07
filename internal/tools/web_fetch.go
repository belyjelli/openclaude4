package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	webFetchHTTPTimeout   = 20 * time.Second
	webFetchMaxBodyBytes  = 2 << 20 // 2 MiB raw response
	webFetchMaxRedirects  = 5
	webFetchDefaultChars  = 80000
	webFetchMaxCharsLimit = 200000
)

// WebFetch downloads a public http(s) URL and returns extracted text for the model (HTML simplified to plain text).
type WebFetch struct{}

func (WebFetch) Name() string      { return "WebFetch" }
func (WebFetch) IsDangerous() bool { return false }

func (WebFetch) Description() string {
	s := "Fetch a web page or text/json document over HTTP(S). Returns plain text (HTML tags stripped). " +
		"Only public URLs; localhost and private IPs are blocked. Body and output size are capped."
	if FirecrawlEnabled() {
		s += " With FIRECRAWL_API_KEY, tries Firecrawl scrape first (better for JS-heavy pages), then falls back to direct HTTP."
	}
	return s
}

func (WebFetch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Absolute http or https URL to fetch",
			},
			"max_chars": map[string]any{
				"type":        "number",
				"description": "Maximum characters of extracted text to return (default 80000, max 200000)",
			},
		},
		"required": []string{"url"},
	}
}

func (WebFetch) Execute(ctx context.Context, args map[string]any) (string, error) {
	raw, _ := args["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if err := ValidateFetchURL(u); err != nil {
		return "", err
	}

	maxChars := webFetchDefaultChars
	if v, ok := args["max_chars"].(float64); ok {
		maxChars = int(v)
	}
	if maxChars <= 0 {
		maxChars = webFetchDefaultChars
	}
	if maxChars > webFetchMaxCharsLimit {
		maxChars = webFetchMaxCharsLimit
	}

	if key := firecrawlAPIKey(); key != "" {
		if md, err := firecrawlScrapeMarkdown(ctx, key, raw); err == nil {
			text, truncNote := truncateRunesWeb(md, maxChars)
			if text != "" {
				return text + truncNote, nil
			}
		}
	}

	client := &http.Client{
		Timeout: webFetchHTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= webFetchMaxRedirects {
				return fmt.Errorf("too many redirects (max %d)", webFetchMaxRedirects)
			}
			if req.URL != nil {
				if err := ValidateFetchURL(req.URL); err != nil {
					return err
				}
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "openclaude4/1.0 (WebFetch)")
	req.Header.Set("Accept", "text/html, text/plain, application/json, */*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, webFetchMaxBodyBytes+1))
	if err != nil {
		return "", err
	}
	if len(rawBody) > webFetchMaxBodyBytes {
		return "", fmt.Errorf("response body exceeds %d bytes", webFetchMaxBodyBytes)
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	truncNote := ""
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	text, err := extractFetchText(rawBody, ct)
	if err != nil {
		return "", fmt.Errorf("decode body: %w", err)
	}

	text = strings.TrimSpace(text)
	if maxChars > 0 && utf8.RuneCountInString(text) > maxChars {
		text = string([]rune(text)[:maxChars])
		truncNote = fmt.Sprintf("\n\n[truncated to %d characters]", maxChars)
	}

	if text == "" {
		return fmt.Sprintf("(empty body after extraction; Content-Type: %s; HTTP %d)", ct, resp.StatusCode), nil
	}
	return text + truncNote, nil
}

// ValidateFetchURL enforces the same http(s) and SSRF-minded rules as [WebFetch].
func ValidateFetchURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("nil url")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http and https URLs are allowed")
	}
	if u.User != nil {
		return fmt.Errorf("URLs with embedded credentials are not allowed")
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return fmt.Errorf("missing host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("localhost URLs are not allowed")
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		if !fetchIPAllowed(ip) {
			return fmt.Errorf("URL host IP is not allowed for fetch")
		}
		return nil
	}
	return fetchResolveHostAllowed(host)
}

func fetchIPAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return false
	}
	return true
}

func fetchResolveHostAllowed(host string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("no addresses for host %q", host)
	}
	for _, ip := range ips {
		if !fetchIPAllowed(ip) {
			return fmt.Errorf("host %q resolves to a disallowed address", host)
		}
	}
	return nil
}

func extractFetchText(raw []byte, contentType string) (string, error) {
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("response is not valid UTF-8; only text responses are supported")
	}
	s := string(raw)

	isHTML := strings.Contains(contentType, "html") ||
		strings.Contains(contentType, "xhtml") ||
		strings.HasPrefix(strings.TrimSpace(s), "<!DOCTYPE") ||
		strings.HasPrefix(strings.TrimSpace(strings.ToLower(s)), "<html")

	if isHTML {
		return htmlToPlainText(strings.NewReader(s))
	}
	return s, nil
}

func htmlToPlainText(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	needsSpace := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style", "noscript", "template", "svg":
				return
			case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "tr", "section", "article":
				if b.Len() > 0 {
					_ = b.WriteByte('\n')
				}
				needsSpace = false
			}
		}
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t == "" {
				return
			}
			if needsSpace && b.Len() > 0 {
				_ = b.WriteByte(' ')
			}
			_, _ = b.WriteString(t)
			needsSpace = true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return strings.TrimSpace(b.String()), nil
}
