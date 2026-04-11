package providerwizard

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/providers"
	"github.com/spf13/viper"
)

type step int

const (
	stChooseProvider step = iota
	stOpenAIBaseMenu
	stOpenAIBaseText
	stOpenAIModelMenu
	stOpenAIModelText
	stOllamaHost
	stOllamaModelMenu
	stOllamaModelText
	stGeminiBaseMenu
	stGeminiBaseText
	stGeminiModelMenu
	stGeminiModelText
	stGitHubBaseMenu
	stGitHubBaseText
	stGitHubModelMenu
	stGitHubModelText
	stOpenRouterBaseMenu
	stOpenRouterBaseText
	stOpenRouterModelMenu
	stOpenRouterModelText
	stFinished
	stCancelled
)

const modelOtherOption = "Other (type model name manually)"

// Wizard is a multi-step provider setup flow with a back stack.
type Wizard struct {
	step      step
	backStack []step

	menuOpts   []string
	menuCursor int

	textLabel   string
	textDefault string
	textHint    string

	// Base menu: index of "from environment" row, or -1 if absent.
	openaiBaseEnvIdx   int
	geminiBaseEnvIdx   int
	githubBaseEnvIdx   int
	openRouterBaseEnvIdx int

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

	openaiModelMenuTags   []string
	geminiModelMenuTags   []string
	githubModelMenuTags   []string
	openRouterModelMenuTags []string

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

func openRouterEnvBase() (envName, value string) {
	if v := strings.TrimSpace(os.Getenv("OPENROUTER_BASE_URL")); v != "" {
		return "OPENROUTER_BASE_URL", strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); v != "" {
		return "OPENAI_BASE_URL", strings.TrimRight(v, "/")
	}
	return "", ""
}

func normalizeOpenRouterBase(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	def := strings.TrimRight(config.DefaultOpenRouterOpenAIBase, "/")
	if s == "" || strings.EqualFold(s, def) {
		return ""
	}
	return s
}

func (w *Wizard) prepare() {
	switch w.step {
	case stChooseProvider:
		w.menuOpts = []string{"openai", "ollama", "gemini", "github", "openrouter"}
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOllamaModelMenu:
		w.menuOpts = append(append([]string{}, w.ollamaTags...), modelOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenAIBaseMenu:
		w.openaiBaseEnvIdx = -1
		opts := []string{"Default (official OpenAI API — no base_url override)"}
		if u := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); u != "" {
			w.openaiBaseEnvIdx = len(opts)
			opts = append(opts, fmt.Sprintf("Use OPENAI_BASE_URL: %s", u))
		}
		opts = append(opts, "Custom base URL…")
		w.menuOpts = opts
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stGeminiBaseMenu:
		w.geminiBaseEnvIdx = -1
		opts := []string{"Default (Google Gemini OpenAI-compatible endpoint)"}
		if u := strings.TrimSpace(os.Getenv("GEMINI_BASE_URL")); u != "" {
			w.geminiBaseEnvIdx = len(opts)
			opts = append(opts, fmt.Sprintf("Use GEMINI_BASE_URL: %s", u))
		}
		opts = append(opts, "Custom base URL…")
		w.menuOpts = opts
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stGitHubBaseMenu:
		w.githubBaseEnvIdx = -1
		opts := []string{"Default (no github.base_url — set GITHUB_BASE_URL for Azure endpoint)"}
		if u := strings.TrimSpace(os.Getenv("GITHUB_BASE_URL")); u != "" {
			w.githubBaseEnvIdx = len(opts)
			opts = append(opts, fmt.Sprintf("Use GITHUB_BASE_URL: %s", u))
		}
		opts = append(opts, "Custom base URL…")
		w.menuOpts = opts
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenRouterBaseMenu:
		w.openRouterBaseEnvIdx = -1
		opts := []string{fmt.Sprintf("Default (%s — no base_url override)", config.DefaultOpenRouterOpenAIBase)}
		if name, u := openRouterEnvBase(); u != "" {
			w.openRouterBaseEnvIdx = len(opts)
			opts = append(opts, fmt.Sprintf("Use %s: %s", name, u))
		}
		opts = append(opts, "Custom base URL…")
		w.menuOpts = opts
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenAIModelMenu:
		w.menuOpts = append(append([]string{}, w.openaiModelMenuTags...), modelOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stGeminiModelMenu:
		w.menuOpts = append(append([]string{}, w.geminiModelMenuTags...), modelOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stGitHubModelMenu:
		w.menuOpts = append(append([]string{}, w.githubModelMenuTags...), modelOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenRouterModelMenu:
		w.menuOpts = append(append([]string{}, w.openRouterModelMenuTags...), modelOtherOption)
		w.menuCursor = clamp(w.menuCursor, 0, len(w.menuOpts)-1)
	case stOpenAIBaseText:
		w.textLabel = "Base URL"
		w.textDefault = ""
		w.textHint = "OpenAI-compatible root, usually ending in /v1"
	case stOpenAIModelText:
		w.textLabel = "Model"
		w.textDefault = defaultOpenAIModel()
		w.textHint = "Empty = use default. OPENAI_API_KEY in environment."
	case stOllamaHost:
		w.textLabel = "Ollama host"
		w.textDefault = ollamaHostDefault()
		w.textHint = "HTTP base without /v1 (e.g. http://127.0.0.1:11434)"
	case stOllamaModelText:
		w.textLabel = "Ollama model tag"
		w.textDefault = defaultOllamaModel()
		w.textHint = "Model name as shown by ollama list"
	case stGeminiBaseText:
		w.textLabel = "Custom base URL"
		w.textDefault = ""
		w.textHint = "OpenAI-compatible Gemini endpoint (often ends with /v1beta/openai)"
	case stGeminiModelText:
		w.textLabel = "Gemini model"
		w.textDefault = defaultGeminiModel()
		w.textHint = "Set GEMINI_API_KEY or GOOGLE_API_KEY in the environment."
	case stGitHubBaseText:
		w.textLabel = "Base URL"
		w.textDefault = ""
		w.textHint = "GitHub Models Azure endpoint (https://<region>.models.ai.azure.com)"
	case stGitHubModelText:
		w.textLabel = "GitHub Models model"
		w.textDefault = defaultGitHubModel()
		w.textHint = "Set GITHUB_TOKEN or GITHUB_PAT in the environment."
	case stOpenRouterBaseText:
		w.textLabel = "Base URL"
		w.textDefault = ""
		w.textHint = fmt.Sprintf("OpenAI-compatible root (e.g. %s); empty = official default", config.DefaultOpenRouterOpenAIBase)
	case stOpenRouterModelText:
		w.textLabel = "OpenRouter model"
		w.textDefault = defaultOpenRouterModel()
		w.textHint = "Set OPENROUTER_KEY or OPENROUTER_API_KEY in the environment."
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

func (w *Wizard) advanceOpenAIModelStep() {
	ctx := context.Background()
	if strings.TrimSpace(w.openaiBase) == "" {
		w.openaiModelMenuTags = providers.WizardDefaultBaseModels(ctx, "openai")
		if len(w.openaiModelMenuTags) > 0 {
			w.push(stOpenAIModelMenu)
			return
		}
	}
	w.push(stOpenAIModelText)
}

func (w *Wizard) advanceGeminiModelStep() {
	ctx := context.Background()
	if strings.TrimSpace(w.geminiBase) == "" {
		w.geminiModelMenuTags = providers.WizardDefaultBaseModels(ctx, "gemini")
		if len(w.geminiModelMenuTags) > 0 {
			w.push(stGeminiModelMenu)
			return
		}
	}
	w.push(stGeminiModelText)
}

func (w *Wizard) advanceGitHubModelStep() {
	ctx := context.Background()
	if strings.TrimSpace(w.githubBase) == "" {
		w.githubModelMenuTags = providers.WizardDefaultBaseModels(ctx, "github")
		if len(w.githubModelMenuTags) > 0 {
			w.push(stGitHubModelMenu)
			return
		}
	}
	w.push(stGitHubModelText)
}

func (w *Wizard) advanceOpenRouterModelStep() {
	ctx := context.Background()
	if strings.TrimSpace(w.orBase) == "" {
		w.openRouterModelMenuTags = providers.WizardDefaultBaseModels(ctx, "openrouter")
		if len(w.openRouterModelMenuTags) > 0 {
			w.push(stOpenRouterModelMenu)
			return
		}
	}
	w.push(stOpenRouterModelText)
}

// StepKind reports how the UI should collect input.
func (w *Wizard) StepKind() StepKind {
	switch w.step {
	case stFinished, stCancelled:
		return StepDone
	case stChooseProvider, stOllamaModelMenu,
		stOpenAIBaseMenu, stGeminiBaseMenu, stGitHubBaseMenu, stOpenRouterBaseMenu,
		stOpenAIModelMenu, stGeminiModelMenu, stGitHubModelMenu, stOpenRouterModelMenu:
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

// IsModelPickMenu is true when choosing a model from a fetched or static list.
func (w *Wizard) IsModelPickMenu() bool {
	switch w.step {
	case stOllamaModelMenu, stOpenAIModelMenu, stGeminiModelMenu, stGitHubModelMenu, stOpenRouterModelMenu:
		return true
	default:
		return false
	}
}

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

// finishedProvider returns which provider path completed at stFinished, or "".
func (w *Wizard) finishedProvider() string {
	if w.step != stFinished {
		return ""
	}
	if w.ollamaHost != "" && w.ollamaModel != "" {
		return "ollama"
	}
	if w.openaiModel != "" {
		return "openai"
	}
	if w.geminiModel != "" {
		return "gemini"
	}
	if w.githubModel != "" {
		return "github"
	}
	if w.orModel != "" {
		return "openrouter"
	}
	return ""
}

// ApplyToViper merges a successfully completed wizard into viper (in-memory),
// matching the YAML emitted by [Wizard.Result]. Returns an error if the wizard
// did not finish successfully, the finished branch cannot be inferred, or
// [config.Validate] fails.
func (w *Wizard) ApplyToViper() error {
	if w.step != stFinished || w.Cancelled() {
		return fmt.Errorf("provider wizard: not completed successfully")
	}
	if strings.TrimSpace(w.result) == "" {
		return fmt.Errorf("provider wizard: empty result")
	}
	switch w.finishedProvider() {
	case "openai":
		viper.Set("provider.name", "openai")
		viper.Set("provider.model", w.openaiModel)
		if b := strings.TrimSpace(w.openaiBase); b != "" {
			viper.Set("provider.base_url", strings.TrimRight(b, "/"))
		} else {
			viper.Set("provider.base_url", "")
		}
	case "ollama":
		viper.Set("provider.name", "ollama")
		viper.Set("ollama.host", w.ollamaHost)
		viper.Set("ollama.model", w.ollamaModel)
	case "gemini":
		viper.Set("provider.name", "gemini")
		viper.Set("gemini.model", w.geminiModel)
		if b := strings.TrimSpace(w.geminiBase); b != "" {
			viper.Set("gemini.base_url", strings.TrimRight(b, "/"))
		} else {
			viper.Set("gemini.base_url", "")
		}
	case "github":
		viper.Set("provider.name", "github")
		viper.Set("github.model", w.githubModel)
		if b := strings.TrimSpace(w.githubBase); b != "" {
			viper.Set("github.base_url", b)
		} else {
			viper.Set("github.base_url", "")
		}
	case "openrouter":
		viper.Set("provider.name", "openrouter")
		viper.Set("openrouter.model", w.orModel)
		if b := strings.TrimSpace(w.orBase); b != "" {
			viper.Set("provider.base_url", strings.TrimRight(b, "/"))
		} else {
			viper.Set("provider.base_url", "")
		}
	default:
		return fmt.Errorf("provider wizard: cannot infer finished provider")
	}
	return config.Validate()
}

// Title for the current step panel / REPL header.
func (w *Wizard) Title() string {
	switch w.step {
	case stChooseProvider:
		return "Provider setup — choose provider"
	case stOpenAIBaseMenu, stOpenAIBaseText, stOpenAIModelMenu, stOpenAIModelText:
		return "OpenAI (or compatible)"
	case stOllamaHost, stOllamaModelMenu, stOllamaModelText:
		return "Ollama"
	case stGeminiBaseMenu, stGeminiBaseText, stGeminiModelMenu, stGeminiModelText:
		return "Gemini"
	case stGitHubBaseMenu, stGitHubBaseText, stGitHubModelMenu, stGitHubModelText:
		return "GitHub Models"
	case stOpenRouterBaseMenu, stOpenRouterBaseText, stOpenRouterModelMenu, stOpenRouterModelText:
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
		return "Applies provider settings to this session and prints YAML to merge into openclaude.yaml for the next start."
	case stOpenAIBaseMenu, stGeminiBaseMenu, stGitHubBaseMenu, stOpenRouterBaseMenu:
		return "Step 2: use the provider default API host, the URL from your environment if set, or a custom base URL."
	case stOllamaModelMenu:
		return "Select a model from your Ollama host, or choose manual entry."
	case stOpenAIModelMenu, stGeminiModelMenu, stGitHubModelMenu, stOpenRouterModelMenu:
		return "Pick a model from the list (official default API), or choose manual entry."
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
			w.push(stOpenAIBaseMenu)
		case "ollama":
			w.push(stOllamaHost)
		case "gemini":
			w.push(stGeminiBaseMenu)
		case "github":
			w.push(stGitHubBaseMenu)
		case "openrouter":
			w.push(stOpenRouterBaseMenu)
		default:
			return fmt.Errorf("unknown provider")
		}
		return nil
	case stOpenAIBaseMenu:
		switch w.menuCursor {
		case 0:
			w.openaiBase = ""
			w.advanceOpenAIModelStep()
		case w.openaiBaseEnvIdx:
			w.openaiBase = strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
			w.advanceOpenAIModelStep()
		default:
			w.push(stOpenAIBaseText)
		}
		return nil
	case stGeminiBaseMenu:
		switch w.menuCursor {
		case 0:
			w.geminiBase = ""
			w.advanceGeminiModelStep()
		case w.geminiBaseEnvIdx:
			w.geminiBase = strings.TrimRight(strings.TrimSpace(os.Getenv("GEMINI_BASE_URL")), "/")
			w.advanceGeminiModelStep()
		default:
			w.push(stGeminiBaseText)
		}
		return nil
	case stGitHubBaseMenu:
		switch w.menuCursor {
		case 0:
			w.githubBase = ""
			w.advanceGitHubModelStep()
		case w.githubBaseEnvIdx:
			w.githubBase = strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_BASE_URL")), "/")
			w.advanceGitHubModelStep()
		default:
			w.push(stGitHubBaseText)
		}
		return nil
	case stOpenRouterBaseMenu:
		switch w.menuCursor {
		case 0:
			w.orBase = ""
			w.advanceOpenRouterModelStep()
		case w.openRouterBaseEnvIdx:
			_, v := openRouterEnvBase()
			w.orBase = normalizeOpenRouterBase(v)
			w.advanceOpenRouterModelStep()
		default:
			w.push(stOpenRouterBaseText)
		}
		return nil
	case stOllamaModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == modelOtherOption {
			w.push(stOllamaModelText)
			return nil
		}
		w.ollamaModel = opt
		w.result = buildOllamaResult(w.ollamaHost, w.ollamaModel)
		w.step = stFinished
		return nil
	case stOpenAIModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == modelOtherOption {
			w.push(stOpenAIModelText)
			return nil
		}
		w.openaiModel = opt
		w.result = buildOpenAIResult(w.openaiModel, w.openaiBase)
		w.step = stFinished
		return nil
	case stGeminiModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == modelOtherOption {
			w.push(stGeminiModelText)
			return nil
		}
		w.geminiModel = opt
		w.result = buildGeminiResult(w.geminiModel, w.geminiBase)
		w.step = stFinished
		return nil
	case stGitHubModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == modelOtherOption {
			w.push(stGitHubModelText)
			return nil
		}
		w.githubModel = opt
		w.result = buildGitHubResult(w.githubModel, w.githubBase)
		w.step = stFinished
		return nil
	case stOpenRouterModelMenu:
		opt := w.menuOpts[w.menuCursor]
		if opt == modelOtherOption {
			w.push(stOpenRouterModelText)
			return nil
		}
		w.orModel = opt
		w.result = buildOpenRouterResult(w.orModel, w.orBase)
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
	case stOpenAIBaseText:
		w.openaiBase = strings.TrimRight(strings.TrimSpace(s), "/")
		w.advanceOpenAIModelStep()
	case stOpenAIModelText:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.openaiModel = model
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
	case stGeminiBaseText:
		w.geminiBase = strings.TrimRight(strings.TrimSpace(s), "/")
		w.advanceGeminiModelStep()
	case stGeminiModelText:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.geminiModel = model
		w.result = buildGeminiResult(w.geminiModel, w.geminiBase)
		w.step = stFinished
	case stGitHubBaseText:
		w.githubBase = strings.TrimSpace(s)
		w.githubBase = strings.TrimRight(w.githubBase, "/")
		w.advanceGitHubModelStep()
	case stGitHubModelText:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.githubModel = model
		w.result = buildGitHubResult(w.githubModel, w.githubBase)
		w.step = stFinished
	case stOpenRouterBaseText:
		w.orBase = normalizeOpenRouterBase(s)
		w.advanceOpenRouterModelStep()
	case stOpenRouterModelText:
		model := s
		if model == "" {
			model = w.textDefault
		}
		w.orModel = model
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
	return w.ParseModelMenuInput(line)
}

// ParseModelMenuInput maps a line to index 1..N for any model-pick menu.
func (w *Wizard) ParseModelMenuInput(line string) (ok bool) {
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
	base = normalizeOpenRouterBase(base)
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
