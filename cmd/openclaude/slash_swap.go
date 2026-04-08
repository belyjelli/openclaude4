package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/spf13/viper"
)

func effectiveClient(st chatState) core.StreamClient {
	if st.live != nil {
		if c := st.live.Client(); c != nil {
			return c
		}
	}
	return st.client
}

func captureProviderModelKeys() map[string]string {
	keys := []string{
		"provider.name",
		"provider.model",
		"ollama.model",
		"gemini.model",
		"github.model",
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = viper.GetString(k)
	}
	return out
}

func restoreProviderModelKeys(m map[string]string) {
	for k, v := range m {
		viper.Set(k, v)
	}
}

func setViperModelForActiveProvider(model string) {
	model = strings.TrimSpace(model)
	switch strings.ToLower(strings.TrimSpace(config.ProviderName())) {
	case "ollama":
		viper.Set("ollama.model", model)
	case "gemini":
		viper.Set("gemini.model", model)
	case "github":
		viper.Set("github.model", model)
	default:
		viper.Set("provider.model", model)
	}
}

func slashSetModel(st chatState, model string, out io.Writer) error {
	if st.isBusy != nil && st.isBusy() {
		return fmt.Errorf("wait for the current model turn to finish before /model")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		_, _ = fmt.Fprintf(out, "Current model: %s (provider %s)\n", config.Model(), config.ProviderName())
		_, _ = fmt.Fprintln(out, "Usage: /model <model-id>")
		return nil
	}
	snap := captureProviderModelKeys()
	setViperModelForActiveProvider(model)
	if err := config.Validate(); err != nil {
		restoreProviderModelKeys(snap)
		return err
	}
	nc, err := providers.NewStreamClient()
	if err != nil {
		restoreProviderModelKeys(snap)
		return fmt.Errorf("model swap failed: %w (config reverted)", err)
	}
	if st.live != nil {
		st.live.SwapClient(nc)
	}
	_, _ = fmt.Fprintf(out, "(model set to %q for provider %s)\n", config.Model(), config.ProviderName())
	return nil
}

func slashSetProvider(st chatState, prov string, out io.Writer) error {
	if st.isBusy != nil && st.isBusy() {
		return fmt.Errorf("wait for the current model turn to finish before changing provider")
	}
	prov = strings.ToLower(strings.TrimSpace(prov))
	switch prov {
	case "openai", "ollama", "gemini", "github":
	default:
		return fmt.Errorf("unknown provider %q (use openai, ollama, gemini, github)", prov)
	}
	snap := captureProviderModelKeys()
	viper.Set("provider.name", prov)
	if err := config.Validate(); err != nil {
		restoreProviderModelKeys(snap)
		return err
	}
	nc, err := providers.NewStreamClient()
	if err != nil {
		restoreProviderModelKeys(snap)
		return fmt.Errorf("provider swap failed: %w (config reverted)", err)
	}
	if st.live != nil {
		st.live.SwapClient(nc)
	}
	_, _ = fmt.Fprintf(out, "(provider set to %s, model %q)\n", config.ProviderName(), config.Model())
	return nil
}
