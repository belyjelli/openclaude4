package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/providerwizard"
	"github.com/gitlawb/openclaude4/internal/providers"
)

func handleProviderWizard(st chatState, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	client := effectiveClient(st)
	if st.providerWizardIn == nil {
		if st.allowConfigEditorWizard {
			return core.SlashStartProviderWizard{}
		}
		_, _ = fmt.Fprintln(out, "Stdin wizard needs the plain REPL. Current session:")
		printProviderInfoTo(client, out)
		_, _ = fmt.Fprintln(out)
		printProviderSetupGuide(out)
		return nil
	}
	return runProviderInteractiveWizard(st, out, st.providerWizardIn, client)
}

// applyProviderWizardToSession updates viper from a finished wizard, builds a new
// stream client, and swaps the live session client. info is a one-line confirmation
// (empty on error). When out is non-nil, info is also written there on success.
func applyProviderWizardToSession(st chatState, w *providerwizard.Wizard, out io.Writer) (info string, err error) {
	if st.isBusy != nil && st.isBusy() {
		return "", fmt.Errorf("wait for the current model turn to finish before changing provider")
	}
	snap := captureProviderModelKeys()
	if err := w.ApplyToViper(); err != nil {
		restoreProviderModelKeys(snap)
		return "", err
	}
	nc, err := providers.NewStreamClient()
	if err != nil {
		restoreProviderModelKeys(snap)
		return "", err
	}
	if st.live != nil {
		st.live.SwapClient(nc)
	}
	providers.WarmChatModelCache()
	info = fmt.Sprintf("(session updated: provider %s, model %q)", config.ProviderName(), config.Model())
	if out != nil {
		_, _ = fmt.Fprintln(out, info)
	}
	return info, nil
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
export OPENAI_BASE_URL=https://openrouter.ai/api/v1   # default if unset (OPENROUTER_BASE_URL also works)

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

func runProviderInteractiveWizard(st chatState, out io.Writer, in io.Reader, client core.StreamClient) error {
	if in == nil {
		return fmt.Errorf("provider wizard: nil reader")
	}
	r := bufio.NewReader(in)
	w := providerwizard.New()

	_, _ = fmt.Fprintln(out, "=== Provider setup wizard ===")
	_, _ = fmt.Fprintln(out, "Applies to this session and prints YAML to save for the next start.")
	_, _ = fmt.Fprintln(out)
	printProviderInfoTo(client, out)
	_, _ = fmt.Fprintln(out)

	for !w.Finished() {
		switch w.StepKind() {
		case providerwizard.StepMenu:
			_, _ = fmt.Fprintln(out, w.Title())
			if b := w.Body(); strings.TrimSpace(b) != "" {
				_, _ = fmt.Fprintln(out, b)
			}
			for i, opt := range w.MenuOptions() {
				_, _ = fmt.Fprintf(out, "  %d) %s\n", i+1, opt)
			}
			_, _ = fmt.Fprintln(out, w.HintLine())
			_, _ = fmt.Fprint(out, "> ")
			line, err := readWizardLine(r)
			if err != nil {
				return err
			}
			if providerwizard.ParseBackInput(line) {
				if !w.Back() && w.IsProviderMenu() {
					w.Cancel()
				}
				continue
			}
			if w.IsProviderMenu() {
				if strings.TrimSpace(line) == "" {
					w.Cancel()
					continue
				}
				ok, cancel := w.ParseProviderMenuInput(line)
				if cancel {
					w.Cancel()
					continue
				}
				if !ok {
					_, _ = fmt.Fprintln(out, "Try 1–5 or a provider name (openai, ollama, gemini, github, openrouter).")
				}
				continue
			}
			// Base URL menus and model-pick menus are all numbered StepMenu screens.
			if !w.ParseModelMenuInput(line) {
				_, _ = fmt.Fprintf(out, "Try a number 1–%d, or b to go back.\n", len(w.MenuOptions()))
			}

		case providerwizard.StepText:
			_, _ = fmt.Fprintln(out, w.Title())
			if h := w.TextHint(); strings.TrimSpace(h) != "" {
				_, _ = fmt.Fprintln(out, h)
			}
			_, _ = fmt.Fprintln(out, w.HintLine())
			_, _ = fmt.Fprintf(out, "%s [%s]: ", w.TextLabel(), w.TextDefault())
			line, err := readWizardLine(r)
			if err != nil {
				return err
			}
			if providerwizard.ParseBackInput(line) {
				_ = w.Back()
				continue
			}
			if err := w.SubmitText(strings.TrimSpace(line)); err != nil {
				_, _ = fmt.Fprintln(out, err.Error())
			}

		default:
			continue
		}
	}

	if w.Cancelled() {
		_, _ = fmt.Fprintln(out, "(cancelled)")
		return nil
	}
	if st.live != nil {
		if _, err := applyProviderWizardToSession(st, w, out); err != nil {
			_, _ = fmt.Fprintf(out, "Could not apply to session: %v\n", err)
		}
	}
	_, _ = fmt.Fprintln(out, w.Result())
	_, _ = fmt.Fprintln(out, "\nMerge YAML into openclaude.yaml to persist for the next start.")
	return nil
}

func readWizardLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}
