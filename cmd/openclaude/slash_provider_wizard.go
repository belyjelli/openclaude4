package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/spf13/viper"
)

func handleProviderWizard(st chatState, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	client := effectiveClient(st)
	if st.providerWizardIn == nil {
		if st.allowConfigEditorWizard {
			return runProviderWizardOpenEditor(out, client)
		}
		_, _ = fmt.Fprintln(out, "Stdin wizard needs the plain REPL. Current session:")
		printProviderInfoTo(client, out)
		_, _ = fmt.Fprintln(out)
		printProviderSetupGuide(out)
		return nil
	}
	return runProviderInteractiveWizard(out, st.providerWizardIn, client)
}

func runProviderWizardOpenEditor(out io.Writer, client core.StreamClient) error {
	path, err := config.WritableConfigPath()
	if err != nil {
		_, _ = fmt.Fprintf(out, "Could not resolve writable config path: %v\n\n", err)
		printProviderInfoTo(client, out)
		_, _ = fmt.Fprintln(out)
		printProviderSetupGuide(out)
		return nil
	}
	_, _ = fmt.Fprintln(out, "=== Provider setup (config file) ===")
	printProviderInfoTo(client, out)
	_, _ = fmt.Fprintln(out)
	printProviderSetupGuide(out)
	_, _ = fmt.Fprintf(out, "\nOpening %q in your editor (create or edit, save, then restart openclaude).\n", path)
	ed := strings.TrimSpace(os.Getenv("VISUAL"))
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if ed == "" {
		ed = "vi"
	}
	cmd := exec.Command(ed, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q: %w", ed, err)
	}
	_, _ = fmt.Fprintln(out, "(editor closed — restart openclaude after saving config changes)")
	return nil
}

func printProviderSetupGuide(out io.Writer) {
	const guide = `Copy-paste reference (merge into openclaude.yaml or use env vars):

--- OpenAI (or compatible) ---
provider:
  name: openai
  model: "gpt-4o-mini"
  # base_url: "https://api.example.com/v1"   # optional
# openai:
#   api_key: "..."    # prefer: export OPENAI_API_KEY=...

--- Ollama (local) ---
export OPENCLAUDE_PROVIDER=ollama
# optional:
export OLLAMA_HOST=http://127.0.0.1:11434
export OLLAMA_MODEL=llama3.2

provider:
  name: ollama
ollama:
  host: "http://127.0.0.1:11434"
  model: "llama3.2"

--- Gemini (OpenAI-compatible API) ---
export OPENCLAUDE_PROVIDER=gemini
export GEMINI_API_KEY=...    # or GOOGLE_API_KEY

provider:
  name: gemini
gemini:
  model: "gemini-2.0-flash"
  # base_url: ""   # optional override

--- GitHub Models ---
export OPENCLAUDE_PROVIDER=github
export GITHUB_TOKEN=...      # or GITHUB_PAT
# optional:
export GITHUB_MODEL=gpt-4o
export GITHUB_BASE_URL=https://<region>.models.ai.azure.com

provider:
  name: github
github:
  model: "gpt-4o"
  # base_url: ""   # optional Azure endpoint

--- OpenRouter (OpenAI-compatible API) ---
export OPENCLAUDE_PROVIDER=openrouter
export OPENROUTER_KEY=...    # or OPENROUTER_API_KEY
# optional:
export OPENROUTER_MODEL=openai/gpt-4o-mini
export OPENAI_BASE_URL=https://openrouter.ai/api/v1   # default if unset

provider:
  name: openrouter
openrouter:
  model: "openai/gpt-4o-mini"
  # api_key: "..."   # prefer env OPENROUTER_KEY
  # provider: ""     # optional OPENROUTER_PROVIDER for /model catalog filter

Then restart openclaude. Run openclaude doctor to verify.
`
	_, _ = fmt.Fprint(out, guide)
}

func runProviderInteractiveWizard(out io.Writer, in io.Reader, client core.StreamClient) error {
	if in == nil {
		return fmt.Errorf("provider wizard: nil reader")
	}
	r := bufio.NewReader(in)

	_, _ = fmt.Fprintln(out, "=== Provider setup wizard ===")
	_, _ = fmt.Fprintln(out, "This prints recommended YAML/env only. Restart openclaude after editing config.")
	_, _ = fmt.Fprintln(out)
	printProviderInfoTo(client, out)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Choose provider:  1 = openai   2 = ollama   3 = gemini   4 = github   5 = openrouter   (empty = cancel)")
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	switch strings.TrimSpace(line) {
	case "":
		_, _ = fmt.Fprintln(out, "(cancelled)")
		return nil
	case "1":
		return wizardOpenAI(out, r)
	case "2":
		return wizardOllama(out, r)
	case "3":
		return wizardGemini(out, r)
	case "4":
		return wizardGitHub(out, r)
	case "5":
		return wizardOpenRouter(out, r)
	default:
		_, _ = fmt.Fprintln(out, "Unrecognized choice — try 1, 2, 3, 4, or 5.")
		return nil
	}
}

func readWizardLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}

func wizardOpenAI(out io.Writer, r *bufio.Reader) error {
	defModel := "gpt-4o-mini"
	if config.ProviderName() == "openai" {
		if m := strings.TrimSpace(config.Model()); m != "" {
			defModel = m
		}
	}
	_, _ = fmt.Fprintf(out, "Model [%s]: ", defModel)
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(line)
	if model == "" {
		model = defModel
	}
	_, _ = fmt.Fprint(out, "Base URL (empty = default api.openai.com / SDK default): ")
	line, err = readWizardLine(r)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(line)

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "provider:\n  name: openai\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Add to openclaude.yaml (or set env equivalents). Use OPENAI_API_KEY in the environment.")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprint(out, b.String())
	_, _ = fmt.Fprintln(out)
	return nil
}

func wizardGitHub(out io.Writer, r *bufio.Reader) error {
	defModel := "gpt-4o"
	if config.ProviderName() == "github" {
		if m := strings.TrimSpace(config.GitHubModelsModel()); m != "" {
			defModel = m
		}
	}
	_, _ = fmt.Fprintf(out, "GitHub Models model [%s]: ", defModel)
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(line)
	if model == "" {
		model = defModel
	}
	_, _ = fmt.Fprint(out, "Base URL (empty = omit; use https://<region>.models.ai.azure.com if needed): ")
	line, err = readWizardLine(r)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(line)
	base = strings.TrimRight(base, "/")

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "provider:\n  name: github\ngithub:\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Set GITHUB_TOKEN or GITHUB_PAT in the environment, then merge YAML or use:")
	_, _ = fmt.Fprintf(out, "  export OPENCLAUDE_PROVIDER=github\n  export GITHUB_MODEL=%q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(out, "  export GITHUB_BASE_URL=%q\n", base)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "YAML snippet:")
	_, _ = fmt.Fprint(out, b.String())
	_, _ = fmt.Fprintln(out)
	return nil
}

func wizardOllama(out io.Writer, r *bufio.Reader) error {
	defHost := ollamaHostForWizard()
	_, _ = fmt.Fprintf(out, "Ollama host [%s]: ", defHost)
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	host := strings.TrimSpace(line)
	if host == "" {
		host = defHost
	}
	host = strings.TrimRight(host, "/")

	defModel := "llama3.2"
	if config.ProviderName() == "ollama" {
		if m := strings.TrimSpace(config.OllamaModel()); m != "" {
			defModel = m
		}
	}
	_, _ = fmt.Fprintf(out, "Ollama model tag [%s]: ", defModel)
	line, err = readWizardLine(r)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(line)
	if model == "" {
		model = defModel
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "provider:\n  name: ollama\nollama:\n  host: %q\n  model: %q\n", host, model)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Merge into openclaude.yaml, or run with:")
	_, _ = fmt.Fprintf(out, "  export OPENCLAUDE_PROVIDER=ollama\n  export OLLAMA_HOST=%q\n  export OLLAMA_MODEL=%q\n", host, model)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "YAML snippet:")
	_, _ = fmt.Fprint(out, b.String())
	_, _ = fmt.Fprintln(out)
	return nil
}

func wizardOpenRouter(out io.Writer, r *bufio.Reader) error {
	defModel := "openai/gpt-4o-mini"
	if config.ProviderName() == "openrouter" {
		if m := strings.TrimSpace(config.OpenRouterModel()); m != "" {
			defModel = m
		}
	}
	_, _ = fmt.Fprintf(out, "OpenRouter model [%s]: ", defModel)
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(line)
	if model == "" {
		model = defModel
	}
	defBase := strings.TrimRight(config.DefaultOpenRouterOpenAIBase, "/")
	_, _ = fmt.Fprintf(out, "Base URL (empty = %s): ", defBase)
	line, err = readWizardLine(r)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(line)
	base = strings.TrimRight(base, "/")

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Set OPENROUTER_KEY or OPENROUTER_API_KEY in the environment, then merge YAML or use:")
	_, _ = fmt.Fprintf(out, "  export OPENCLAUDE_PROVIDER=openrouter\n  export OPENROUTER_MODEL=%q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(out, "  export OPENAI_BASE_URL=%q\n", base)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "YAML snippet:")
	if base != "" {
		_, _ = fmt.Fprintf(out, "provider:\n  name: openrouter\n  base_url: %q\nopenrouter:\n  model: %q\n", base, model)
	} else {
		_, _ = fmt.Fprintf(out, "provider:\n  name: openrouter\nopenrouter:\n  model: %q\n", model)
	}
	_, _ = fmt.Fprintln(out)
	return nil
}

// ollamaHostForWizard returns ollama.host from config (no /v1 suffix), for YAML examples.
func ollamaHostForWizard() string {
	raw := strings.TrimSpace(viper.GetString("ollama.host"))
	if raw == "" {
		return "http://127.0.0.1:11434"
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/v1") {
		return strings.TrimSuffix(raw, "/v1")
	}
	return raw
}

func wizardGemini(out io.Writer, r *bufio.Reader) error {
	defModel := "gemini-2.0-flash"
	if config.ProviderName() == "gemini" {
		if m := strings.TrimSpace(config.GeminiModel()); m != "" {
			defModel = m
		}
	}
	_, _ = fmt.Fprintf(out, "Gemini model [%s]: ", defModel)
	line, err := readWizardLine(r)
	if err != nil {
		return err
	}
	model := strings.TrimSpace(line)
	if model == "" {
		model = defModel
	}
	_, _ = fmt.Fprint(out, "Custom base URL (empty = Google default OpenAI-compat endpoint): ")
	line, err = readWizardLine(r)
	if err != nil {
		return err
	}
	base := strings.TrimSpace(line)

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "provider:\n  name: gemini\ngemini:\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Set GEMINI_API_KEY or GOOGLE_API_KEY in the environment, then merge YAML or use:")
	_, _ = fmt.Fprintf(out, "  export OPENCLAUDE_PROVIDER=gemini\n  export GEMINI_MODEL=%q\n", model)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "YAML snippet:")
	_, _ = fmt.Fprint(out, b.String())
	_, _ = fmt.Fprintln(out)
	return nil
}
