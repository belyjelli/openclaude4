package providerwizard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/spf13/viper"
)

type step int

const (
	stChooseProvider step = iota
	stOpenAIModel
	stOpenAIBase
	stOllamaHost
	stOllamaModelMenu
	stOllamaModelText
	stGeminiModel
	stGeminiBase
	stGitHubModel
	stGitHubBase
	stOpenRouterModel
	stOpenRouterBase
	stFinished
	stCancelled
)

const ollamaOtherOption = "Other (type model name manually)"

// Wizard is a multi-step provider setup flow with a back stack.
type Wizard struct {
	step      step
	backStack []step

	menuOpts   []string
	menuCursor int

	textLabel   string
	textDefault string
	textHint    string

	// Collected
	openaiModel string
	openaiBase  string

	ollamaHost  string
	ollamaModel string
	ollamaTags  []string

	geminiModel string
	geminiBase  string

	githubModel string
	githubBase  string

	orModel string
	orBase  string

	result string
}

// New creates a wizard at the provider selection step.
func New() *Wizard {
	w := &Wizard{step: stChooseProvider}
	w.prepare()
	return w
}

func ollamaHostDefault() string {
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

func defaultOpenAIModel() string {
	def := "gpt-4o-mini"
	if config.ProviderName() == "openai" {
		if m := strings.TrimSpace(config.Model()); m != "" {
			return m
		}
	}
	return def
}

func defaultOllamaModel() string {
	def := "llama3.2"
	if config.ProviderName() == "ollama" {
		if m := strings.TrimSpace(config.OllamaModel()); m != "" {
			return m
		}
	}
	return def
}

func defaultGeminiModel() string {
	def := "gemini-2.0-flash"
	if config.ProviderName() == "gemini" {
		if m := strings.TrimSpace(config.GeminiModel()); m != "" {
			return m
		}
	}
	return def
}

func defaultGitHubModel() string {
	def := "gpt-4o"
	if config.ProviderName() == "github" {
		if m := strings.TrimSpace(config.GitHubModelsModel()); m != "" {
			return m
		}
	}
	return def
}

func defaultOpenRouterModel() string {
	def := "openai/gpt-4o-mini"
	if config.ProviderName() == "openrouter" {
		if m := strings.TrimSpace(config.OpenRouterModel()); m != "" {
			return m
		}
	}
	return def
}

func (w *Wizard) prepare() {
	switch w.step {
	case stChooseProvider:
		w.menuOpts = []string{"openai", "ollama", "gemini", "github", "openrouter"}
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOllamaModelMenu:
		w.menuOpts = append(append([]string{}, w.ollamaTags...), ollamaOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenAIModel:
		w.textLabel = "Model"
		w.textDefault = defaultOpenAIModel()
		w.textHint = "Empty = use default. OPENAI_API_KEY in environment."
	case stOpenAIBase:
		w.textLabel = "Base URL"
		w.textDefault = ""
		w.textHint = "Empty = default api.openai.com / SDK default"
	case stOllamaHost:
		w.textLabel = "Ollama host"
		w.textDefault = ollamaHostDefault()
		w.textHint = "HTTP base without /v1 (e.g. http://127.0.0.1:11434)"
	case stOllamaModelText:
		w.textLabel = "Ollama model tag"
		w.textDefault = defaultOllamaModel()
		w.textHint = "Model name as shown by ollama list"
	case stGeminiModel:
		w.textLabel = "Gemini model"
		w.textDefault = defaultGeminiModel()
		w.textHint = "Set GEMINI_API_KEY or GOOGLE_API_KEY in the environment."
	case stGeminiBase:
		w.textLabel = "Custom base URL"
		w.textDefault = ""
		w.textHint = "Empty = Google default OpenAI-compat endpoint"
	case stGitHubModel:
		w.textLabel = "GitHub Models model"
		w.textDefault = defaultGitHubModel()
		w.textHint = "Set GITHUB_TOKEN or GITHUB_PAT in the environment."
	case stGitHubBase:
		w.textLabel = "Base URL"
		w.textDefault = ""
		w.textHint = "Empty = omit; use https://<region>.models.ai.azure.com if needed"
	case stOpenRouterModel:
		w.textLabel = "OpenRouter model"
		w.textDefault = defaultOpenRouterModel()
		w.textHint = "Set OPENROUTER_KEY or OPENROUTER_API_KEY in the environment."
	case stOpenRouterBase:
		w.textLabel = "Base URL"
		w.textDefault = strings.TrimRight(config.DefaultOpenRouterOpenAIBase, "/")
		w.textHint = "Empty = OpenRouter default"
	default:
		w.menuOpts = nil
	}
}

func clamp(i, lo, hi int) int {
	if i < lo {
		return lo
	}
	if i > hi {
		return hi
	}
	return i
}

func (w *Wizard) push(next step) {
	w.backStack = append(w.backStack, w.step)
	w.step = next
	w.prepare()
}

// StepKind reports how the UI should collect input.
func (w *Wizard) StepKind() StepKind {
	switch w.step {
	case stFinished, stCancelled:
		return StepDone
	case stChooseProvider, stOllamaModelMenu:
		return StepMenu
	default:
		return StepText
	}
}

// AtRoot is true on the first provider menu (Esc cancels whole wizard).
func (w *Wizard) AtRoot() bool {
	return w.step == stChooseProvider && len(w.backStack) == 0
}

// IsProviderMenu is true on the initial provider list step.
func (w *Wizard) IsProviderMenu() bool { return w.step == stChooseProvider }

// IsOllamaModelMenu is true when picking a tag from /api/tags.
func (w *Wizard) IsOllamaModelMenu() bool { return w.step == stOllamaModelMenu }

// Finished is true when the flow ended (success or cancel).
func (w *Wizard) Finished() bool {
	return w.step == stFinished || w.step == stCancelled
}

// Cancelled is true if the user aborted.
func (w *Wizard) Cancelled() bool {
	return w.step == stCancelled
}

// Result is non-empty after a successful finish (YAML + instructions).
func (w *Wizard) Result() string {
	return w.result
}

// Title for the current step panel / REPL header.
func (w *Wizard) Title() string {
	switch w.step {
	case stChooseProvider:
		return "Provider setup — choose provider"
	case stOpenAIModel, stOpenAIBase:
		return "OpenAI (or compatible)"
	case stOllamaHost, stOllamaModelMenu, stOllamaModelText:
		return "Ollama"
	case stGeminiModel, stGeminiBase:
		return "Gemini"
	case stGitHubModel, stGitHubBase:
		return "GitHub Models"
	case stOpenRouterModel, stOpenRouterBase:
		return "OpenRouter"
	case stFinished:
		return "Done"
	case stCancelled:
		return "Cancelled"
	default:
		return "Provider setup"
	}
}

// Body is descriptive text for the current step.
func (w *Wizard) Body() string {
	switch w.step {
	case stChooseProvider:
		return "This wizard prints recommended YAML/env only. Restart openclaude after editing config."
	case stOllamaModelMenu:
		return "Select a model from your Ollama host, or choose manual entry."
	default:
		return ""
	}
}

// HintLine is keyboard help for TUI / REPL.
func (w *Wizard) HintLine() string {
	if w.StepKind() == StepMenu {
		if w.AtRoot() {
			return "↑↓ (or numbers in REPL) · Enter · b back · esc cancel (REPL: empty line also cancels)"
		}
		return "↑↓ navigate · Enter confirm · b back · esc cancel"
	}
	if w.StepKind() == StepText {
		return "Enter = submit (empty uses default where shown) · b back · esc cancel"
	}
	return ""
}

// MenuOptions returns selectable rows for menu steps.
func (w *Wizard) MenuOptions() []string {
	return w.menuOpts
}

// MenuCursor is the selected index for menu steps.
func (w *Wizard) MenuCursor() int {
	return w.menuCursor
}

// MenuMove changes selection by delta (-1 / +1).
func (w *Wizard) MenuMove(delta int) {
	if len(w.menuOpts) == 0 {
		return
	}
	w.menuCursor = (w.menuCursor + delta + len(w.menuOpts)) % len(w.menuOpts)
}

// TextLabel for text steps.
func (w *Wizard) TextLabel() string { return w.textLabel }

// TextDefault for text steps (placeholder / empty-submit default).
func (w *Wizard) TextDefault() string { return w.textDefault }

// TextHint for text steps.
func (w *Wizard) TextHint() string { return w.textHint }

// SelectMenuIndex chooses the option at i. Returns error if out of range.
func (w *Wizard) SelectMenuIndex(i int) error {
	if w.StepKind() != StepMenu {
		return fmt.Errorf("not a menu step")
	}
	if i < 0 || i >= len(w.menuOpts) {
		return fmt.Errorf("invalid index")
	}
	w.menuCursor = i
	return w.activateMenuSelection()
}

// SelectCurrentMenu chooses the highlighted option (Enter in TUI).
func (w *Wizard) SelectCurrentMenu() error {
	return w.SelectMenuIndex(w.menuCursor)
}

// activateMenuSelection applies the current menuCursor (internal).
func (w *Wizard) activateMenuSelection() error {
	switch w.step {
	case stChooseProvider:
		p := strings.ToLower(strings.TrimSpace(w.menuOpts[w.menuCursor]))
		switch p {
		case "openai":
			w.push(stOpenAIModel)
		case "ollama":
			w.push(stOllamaHost)
		case "gemini":
			w.push(stGeminiModel)
		case "github":
			w.push(stGitHubModel)
		case "openrouter":
			w.push(stOpenRouterModel)
		default:
			return fmt.Errorf("unknown provider")
		}
		return nil
	case stOllamaModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == ollamaOtherOption {
			w.push(stOllamaModelText)
			return nil
		}
		w.ollamaModel = opt
		w.result = buildOllamaResult(w.ollamaHost, w.ollamaModel)
		w.step = stFinished
		return nil
	default:
		return fmt.Errorf("unknown menu step")
	}
}

// SubmitText applies input for a text step. Empty uses default when defined.
func (w *Wizard) SubmitText(s string) error {
	if w.StepKind() != StepText {
		return fmt.Errorf("not a text step")
	}
	s = strings.TrimSpace(s)
	switch w.step {
	case stOpenAIModel:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.openaiModel = model
		w.push(stOpenAIBase)
	case stOpenAIBase:
		w.openaiBase = strings.TrimSpace(s)
		w.result = buildOpenAIResult(w.openaiModel, w.openaiBase)
		w.step = stFinished
	case stOllamaHost:
		host := s
		if host == "" {
			host = w.textDefault
		}
		host = strings.TrimRight(host, "/")
		w.ollamaHost = host
		tags, err := ListOllamaModelTags(host)
		if err == nil && len(tags) > 0 {
			w.ollamaTags = tags
			w.push(stOllamaModelMenu)
		} else {
			w.push(stOllamaModelText)
		}
	case stOllamaModelText:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.ollamaModel = model
		w.result = buildOllamaResult(w.ollamaHost, w.ollamaModel)
		w.step = stFinished
	case stGeminiModel:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.geminiModel = model
		w.push(stGeminiBase)
	case stGeminiBase:
		w.geminiBase = strings.TrimSpace(s)
		w.result = buildGeminiResult(w.geminiModel, w.geminiBase)
		w.step = stFinished
	case stGitHubModel:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.githubModel = model
		w.push(stGitHubBase)
	case stGitHubBase:
		w.githubBase = strings.TrimSpace(s)
		w.githubBase = strings.TrimRight(w.githubBase, "/")
		w.result = buildGitHubResult(w.githubModel, w.githubBase)
		w.step = stFinished
	case stOpenRouterModel:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.orModel = model
		w.push(stOpenRouterBase)
	case stOpenRouterBase:
		base := strings.TrimSpace(s)
		base = strings.TrimRight(base, "/")
		if base == "" {
			base = ""
		}
		w.orBase = base
		w.result = buildOpenRouterResult(w.orModel, w.orBase)
		w.step = stFinished
	default:
		return fmt.Errorf("unknown text step")
	}
	return nil
}

// Back pops one level. False if already at provider menu root.
func (w *Wizard) Back() bool {
	if len(w.backStack) == 0 {
		return false
	}
	n := len(w.backStack) - 1
	w.step = w.backStack[n]
	w.backStack = w.backStack[:n]
	w.prepare()
	return true
}

// Cancel aborts the wizard.
func (w *Wizard) Cancel() {
	w.step = stCancelled
	w.backStack = nil
	w.menuOpts = nil
}

// ParseProviderMenuInput maps REPL tokens to provider selection (1–5 or name). Empty string means cancel.
func (w *Wizard) ParseProviderMenuInput(line string) (ok bool, cancel bool) {
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return false, true
	}
	if n, err := strconv.Atoi(line); err == nil {
		if n >= 1 && n <= len(w.menuOpts) {
			_ = w.SelectMenuIndex(n - 1)
			return true, false
		}
		return false, false
	}
	for i, opt := range w.menuOpts {
		if strings.ToLower(opt) == line {
			_ = w.SelectMenuIndex(i)
			return true, false
		}
	}
	return false, false
}

// ParseOllamaMenuInput maps a line to index 1..N for the Ollama model menu.
func (w *Wizard) ParseOllamaMenuInput(line string) (ok bool) {
	line = strings.TrimSpace(line)
	if n, err := strconv.Atoi(line); err == nil {
		if n >= 1 && n <= len(w.menuOpts) {
			return w.SelectMenuIndex(n-1) == nil
		}
	}
	return false
}

// ParseBackInput returns true if line means “go back” (menu or text step).
func ParseBackInput(line string) bool {
	s := strings.TrimSpace(strings.ToLower(line))
	return s == "b" || s == "back"
}

func buildOpenAIResult(model, base string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Add to openclaude.yaml (or set env equivalents). Use OPENAI_API_KEY in the environment.\n\n")
	_, _ = fmt.Fprintf(&b, "provider:\n  name: openai\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	return b.String()
}

func buildOllamaResult(host, model string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Merge into openclaude.yaml, or run with:\n")
	_, _ = fmt.Fprintf(&b, "  export OPENCLAUDE_PROVIDER=ollama\n  export OLLAMA_HOST=%q\n  export OLLAMA_MODEL=%q\n\n", host, model)
	_, _ = fmt.Fprintf(&b, "YAML snippet:\n")
	_, _ = fmt.Fprintf(&b, "provider:\n  name: ollama\nollama:\n  host: %q\n  model: %q\n", host, model)
	return b.String()
}

func buildGeminiResult(model, base string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Set GEMINI_API_KEY or GOOGLE_API_KEY in the environment, then merge YAML or use:\n")
	_, _ = fmt.Fprintf(&b, "  export OPENCLAUDE_PROVIDER=gemini\n  export GEMINI_MODEL=%q\n\n", model)
	_, _ = fmt.Fprintf(&b, "YAML snippet:\n")
	_, _ = fmt.Fprintf(&b, "provider:\n  name: gemini\ngemini:\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	return b.String()
}

func buildGitHubResult(model, base string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Set GITHUB_TOKEN or GITHUB_PAT in the environment, then merge YAML or use:\n")
	_, _ = fmt.Fprintf(&b, "  export OPENCLAUDE_PROVIDER=github\n  export GITHUB_MODEL=%q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  export GITHUB_BASE_URL=%q\n", base)
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintf(&b, "YAML snippet:\n")
	_, _ = fmt.Fprintf(&b, "provider:\n  name: github\ngithub:\n  model: %q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  base_url: %q\n", base)
	}
	return b.String()
}

func buildOpenRouterResult(model, base string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "Set OPENROUTER_KEY or OPENROUTER_API_KEY in the environment, then merge YAML or use:\n")
	_, _ = fmt.Fprintf(&b, "  export OPENCLAUDE_PROVIDER=openrouter\n  export OPENROUTER_MODEL=%q\n", model)
	if base != "" {
		_, _ = fmt.Fprintf(&b, "  export OPENAI_BASE_URL=%q\n", base)
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintf(&b, "YAML snippet:\n")
	if base != "" {
		_, _ = fmt.Fprintf(&b, "provider:\n  name: openrouter\n  base_url: %q\nopenrouter:\n  model: %q\n", base, model)
	} else {
		_, _ = fmt.Fprintf(&b, "provider:\n  name: openrouter\nopenrouter:\n  model: %q\n", model)
	}
	return b.String()
}
