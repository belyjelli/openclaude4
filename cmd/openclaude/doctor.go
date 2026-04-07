package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print environment and provider diagnostics",
	Run:   runDoctor,
}

func runDoctor(_ *cobra.Command, _ []string) {
	_, _ = fmt.Fprintf(os.Stdout, "openclaude %s (%s)\n", version, commit)
	_, _ = fmt.Fprintf(os.Stdout, "Go runtime: %s\n", runtime.Version())

	if err := config.Validate(); err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Config validation: %v\n", err)
	}

	if _, err := exec.LookPath("rg"); err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "ripgrep (rg): not found on PATH (Grep tool uses Go regexp only)\n")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "ripgrep (rg): found\n")
	}

	_, _ = fmt.Fprintf(os.Stdout, "Active provider: %s\n", config.ProviderName())
	_, _ = fmt.Fprintf(os.Stdout, "Model: %s\n", config.Model())

	switch config.ProviderName() {
	case "ollama":
		_, _ = fmt.Fprintf(os.Stdout, "Ollama API base: %s\n", config.OllamaChatBase())
	case "gemini":
		_, _ = fmt.Fprintf(os.Stdout, "Gemini OpenAI-compat base: %s\n", config.GeminiBaseURL())
	default:
		if b := config.BaseURL(); b != "" {
			_, _ = fmt.Fprintf(os.Stdout, "OpenAI base URL: %s\n", b)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s\n", providers.PingProviderBestEffort())

	if _, err := providers.NewStreamClient(); err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Client: error — %v\n", err)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Client: configuration OK for chat\n")
	}
}
