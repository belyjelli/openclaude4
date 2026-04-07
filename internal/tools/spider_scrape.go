package tools

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	spiderScrapeTimeout = 60 * time.Second
	spiderMaxChars      = webFetchMaxCharsLimit
	spiderDefaultChars  = webFetchDefaultChars
)

// SpiderOnPath reports whether the spider CLI (spider-rs spider_cli) is available.
func SpiderOnPath() (path string, ok bool) {
	p, err := exec.LookPath("spider")
	return p, err == nil
}

func registerSpiderIfAvailable(r *Registry) {
	if _, err := exec.LookPath("spider"); err != nil {
		return
	}
	r.Register(SpiderScrape{})
}

// SpiderScrape runs the external spider CLI in scrape mode for one URL (text/markdown output for LLMs).
// Registered only when `spider` is on PATH; see https://github.com/spider-rs/spider (cargo install spider_cli).
type SpiderScrape struct{}

func (SpiderScrape) Name() string      { return "SpiderScrape" }
func (SpiderScrape) IsDangerous() bool { return false }

func (SpiderScrape) Description() string {
	return "Scrape one public http(s) URL using the **spider** CLI on PATH (Rust spider_cli: `cargo install spider_cli`). " +
		"Uses `spider --url … --return-format <fmt> scrape --output-html`. Same URL restrictions as WebFetch. " +
		"If this tool is missing, install spider or use WebFetch/WebSearch instead."
}

func (SpiderScrape) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Absolute http or https URL to scrape",
			},
			"return_format": map[string]any{
				"type":        "string",
				"description": "spider --return-format: text, markdown, commonmark, raw, or xml (default: text)",
			},
			"max_chars": map[string]any{
				"type":        "number",
				"description": "Maximum characters of stdout to return (default 80000, max 200000)",
			},
		},
		"required": []string{"url"},
	}
}

func (SpiderScrape) Execute(ctx context.Context, args map[string]any) (string, error) {
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

	format := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["return_format"])))
	switch format {
	case "":
		format = "text"
	case "text", "markdown", "commonmark", "raw", "xml":
	default:
		return "", fmt.Errorf("return_format must be text, markdown, commonmark, raw, or xml")
	}

	maxChars := spiderDefaultChars
	if v, ok := args["max_chars"].(float64); ok {
		maxChars = int(v)
	}
	if maxChars <= 0 {
		maxChars = spiderDefaultChars
	}
	if maxChars > spiderMaxChars {
		maxChars = spiderMaxChars
	}

	spiderPath, err := exec.LookPath("spider")
	if err != nil {
		return "", fmt.Errorf("spider CLI not on PATH (install: cargo install spider_cli)")
	}

	runCtx, cancel := context.WithTimeout(ctx, spiderScrapeTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, spiderPath,
		"--url", u.String(),
		"--return-format", format,
		"scrape",
		"--output-html",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("spider: %w (%s)", err, msg)
		}
		return "", fmt.Errorf("spider: %w", err)
	}

	out := stdout.Bytes()
	if !utf8.Valid(out) {
		return "", fmt.Errorf("spider output is not valid UTF-8")
	}
	text := strings.TrimSpace(string(out))
	truncNote := ""
	if maxChars > 0 && utf8.RuneCountInString(text) > maxChars {
		text = string([]rune(text)[:maxChars])
		truncNote = fmt.Sprintf("\n\n[truncated to %d characters]", maxChars)
	}
	if text == "" {
		return "(spider produced empty stdout)", nil
	}
	return text + truncNote, nil
}
