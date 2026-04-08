package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/providers"
)

// PrintDoctorReport writes the same diagnostics as the doctor subcommand.
func PrintDoctorReport(w io.Writer, ver, cmt string) {
	if w == nil {
		w = io.Discard
	}
	_, _ = fmt.Fprintf(w, "openclaude %s (%s)\n", ver, cmt)
	_, _ = fmt.Fprintf(w, "Go runtime: %s\n", runtime.Version())

	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(w, "Config validation: %v\n", err)
	}

	if _, err := exec.LookPath("rg"); err != nil {
		_, _ = fmt.Fprintf(w, "ripgrep (rg): not found on PATH (Grep tool uses Go regexp only)\n")
	} else {
		_, _ = fmt.Fprintln(w, "ripgrep (rg): found")
	}

	if p, err := exec.LookPath("spider"); err != nil {
		_, _ = fmt.Fprintf(w, "spider (spider_cli): not found on PATH (optional SpiderScrape tool not registered; cargo install spider_cli)\n")
	} else {
		_, _ = fmt.Fprintf(w, "spider (spider_cli): found at %s — SpiderScrape tool enabled\n", p)
	}

	_, _ = fmt.Fprintf(w, "Active provider: %s\n", config.ProviderName())
	_, _ = fmt.Fprintf(w, "Model: %s\n", config.Model())

	switch config.ProviderName() {
	case "ollama":
		_, _ = fmt.Fprintf(w, "Ollama API base: %s\n", config.OllamaChatBase())
	case "gemini":
		_, _ = fmt.Fprintf(w, "Gemini OpenAI-compat base: %s\n", config.GeminiBaseURL())
	case "github":
		_, _ = fmt.Fprintf(w, "GitHub Models base: %s\n", config.GitHubModelsBaseURL())
	default:
		if b := config.BaseURL(); b != "" {
			_, _ = fmt.Fprintf(w, "OpenAI base URL: %s\n", b)
		}
	}

	_, _ = fmt.Fprintf(w, "%s\n", providers.PingProviderBestEffort())

	mcpSrv := config.MCPServers()
	if len(mcpSrv) == 0 {
		_, _ = fmt.Fprintln(w, "MCP (config): no servers in mcp.servers")
	} else {
		_, _ = fmt.Fprintf(w, "MCP (config): %d server(s)\n", len(mcpSrv))
		for _, s := range mcpSrv {
			cmd0 := ""
			if len(s.Command) > 0 {
				cmd0 = s.Command[0]
			}
			ap := s.Approval
			if ap == "" {
				ap = "ask"
			}
			_, _ = fmt.Fprintf(w, "  - %s: argv0=%q approval=%s\n", s.Name, cmd0, ap)
		}
	}

	if _, err := providers.NewStreamClient(); err != nil {
		_, _ = fmt.Fprintf(w, "Client: error — %v\n", err)
	} else {
		_, _ = fmt.Fprintln(w, "Client: configuration OK for chat")
	}
}
