package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/providers/openaicomp"
)

const chatModelCacheTTL = 45 * time.Second

var chatModelCache struct {
	mu       sync.Mutex
	key      string
	at       time.Time
	ids      []string
	warnings []string
}

// WarmChatModelCache refreshes the in-memory model list in the background (non-blocking).
func WarmChatModelCache() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		_, _, _ = FetchChatModelIDs(ctx)
	}()
}

// DefaultChatModelIDs returns built-in model id suggestions when live listing is unavailable.
func DefaultChatModelIDs() []string {
	return staticChatModelIDs(config.ProviderName())
}

// CachedChatModelIDsForSuggest returns a cached list if fresh, otherwise static defaults for the active provider.
func CachedChatModelIDsForSuggest() []string {
	chatModelCache.mu.Lock()
	defer chatModelCache.mu.Unlock()
	if len(chatModelCache.ids) > 0 && time.Since(chatModelCache.at) < chatModelCacheTTL &&
		chatModelCache.key == chatModelConfigKey() {
		out := make([]string, len(chatModelCache.ids))
		copy(out, chatModelCache.ids)
		return out
	}
	return staticChatModelIDs(config.ProviderName())
}

// FetchChatModelIDs returns chat model IDs for the active provider (network when cache is stale).
// Warnings are non-fatal hints (e.g. fallback list used).
func FetchChatModelIDs(ctx context.Context) ([]string, []string, error) {
	key := chatModelConfigKey()
	chatModelCache.mu.Lock()
	if len(chatModelCache.ids) > 0 && time.Since(chatModelCache.at) < chatModelCacheTTL && chatModelCache.key == key {
		out := append([]string(nil), chatModelCache.ids...)
		w := append([]string(nil), chatModelCache.warnings...)
		chatModelCache.mu.Unlock()
		return out, w, nil
	}
	chatModelCache.mu.Unlock()

	ids, warns, err := fetchChatModelIDsUncached(ctx)
	if err != nil {
		return nil, warns, err
	}
	sort.Strings(ids)
	ids = dedupeSorted(ids)

	chatModelCache.mu.Lock()
	chatModelCache.key = key
	chatModelCache.at = time.Now()
	chatModelCache.ids = append([]string(nil), ids...)
	chatModelCache.warnings = append([]string(nil), warns...)
	chatModelCache.mu.Unlock()

	return ids, warns, nil
}

func chatModelConfigKey() string {
	p := strings.ToLower(strings.TrimSpace(config.ProviderName()))
	switch p {
	case "ollama":
		return p + "|" + config.OllamaChatBase()
	case "gemini":
		return p + "|" + fmt.Sprintf("%d", len(config.GeminiAPIKey()))
	case "github":
		return p + "|" + config.GitHubModelsBaseURL() + "|" + fmt.Sprintf("%d", len(config.GitHubToken()))
	case "openrouter":
		return p + "|" + fmt.Sprintf("%d", len(config.OpenRouterAPIKey())) + "|" + config.OpenRouterProviderFilter()
	default:
		return p + "|" + config.BaseURL() + "|" + fmt.Sprintf("%d", len(config.EffectiveOpenAICompatAPIKey())) +
			"|or:" + fmt.Sprintf("%d", len(config.OpenRouterAPIKey())) + "|" + config.OpenRouterProviderFilter()
	}
}

func fetchChatModelIDsUncached(ctx context.Context) ([]string, []string, error) {
	var warns []string
	switch strings.ToLower(strings.TrimSpace(config.ProviderName())) {
	case "ollama":
		ids, err := fetchOllamaModelNames(ctx)
		if err != nil {
			warns = append(warns, "Ollama: "+err.Error()+" — showing common local tags; start Ollama or run `ollama pull <model>`.")
			return staticChatModelIDs("ollama"), warns, nil
		}
		if len(ids) == 0 {
			warns = append(warns, "Ollama returned no models — use `ollama pull` or set /model manually.")
			return staticChatModelIDs("ollama"), warns, nil
		}
		return ids, warns, nil
	case "gemini":
		key := config.GeminiAPIKey()
		if key == "" {
			return nil, warns, openaicomp.ErrMissingGeminiKey
		}
		ids, err := fetchGeminiGenerateContentModels(ctx, key)
		if err != nil {
			warns = append(warns, "Gemini: "+err.Error()+" — showing common Gemini model IDs.")
			return staticChatModelIDs("gemini"), warns, nil
		}
		return ids, warns, nil
	case "github":
		tok := config.GitHubToken()
		if tok == "" {
			return nil, warns, openaicomp.ErrMissingGitHubToken
		}
		base := strings.TrimSpace(config.GitHubModelsBaseURL())
		if base == "" {
			warns = append(warns, "GITHUB_BASE_URL is unset — showing common GitHub Models IDs; set the Azure endpoint from GitHub docs.")
			return staticChatModelIDs("github"), warns, nil
		}
		ids, err := fetchOpenAICompatModelsList(ctx, strings.TrimRight(base, "/")+"/models", tok)
		if err != nil {
			warns = append(warns, "GitHub Models: "+err.Error()+" — showing common model IDs.")
			return staticChatModelIDs("github"), warns, nil
		}
		return ids, warns, nil
	case "openrouter":
		key := config.OpenRouterAPIKey()
		if key == "" {
			return nil, warns, openaicomp.ErrMissingOpenRouterKey
		}
		filter := config.OpenRouterProviderFilter()
		ids, err := fetchOpenRouterChatModelIDs(ctx, key, filter)
		if err != nil {
			warns = append(warns, "OpenRouter: "+err.Error()+" — showing common OpenRouter model IDs.")
			return staticChatModelIDs("openrouter"), warns, nil
		}
		if len(ids) == 0 {
			if filter != "" {
				warns = append(warns, "OpenRouter: no models matched OPENROUTER_PROVIDER="+filter+".")
			}
			return staticChatModelIDs("openrouter"), warns, nil
		}
		return ids, warns, nil
	default:
		if orKey := config.OpenRouterAPIKey(); orKey != "" {
			filter := config.OpenRouterProviderFilter()
			ids, err := fetchOpenRouterChatModelIDs(ctx, orKey, filter)
			if err != nil {
				warns = append(warns, "OpenRouter: "+err.Error()+" — try OPENROUTER_KEY or fall back below.")
				// Fall through to OpenAI-compat list when OPENAI_API_KEY is also set.
			} else if len(ids) > 0 {
				return ids, warns, nil
			} else if filter != "" {
				warns = append(warns, "OpenRouter: no models matched OPENROUTER_PROVIDER="+filter+".")
			} else {
				warns = append(warns, "OpenRouter: no chat models returned after filtering.")
			}
		}
		k := config.EffectiveOpenAICompatAPIKey()
		if k == "" {
			if config.OpenRouterAPIKey() != "" {
				return staticChatModelIDs("openai"), warns, nil
			}
			if config.BaseURLLooksLikeOpenRouter(config.BaseURL()) {
				return nil, warns, openaicomp.ErrMissingOpenRouterOrOpenAIKey
			}
			return nil, warns, openaicomp.ErrMissingAPIKey
		}
		listURL, err := openAICompatModelsURL()
		if err != nil {
			return nil, warns, err
		}
		ids, err := fetchOpenAICompatModelsList(ctx, listURL, k)
		if err != nil {
			warns = append(warns, "OpenAI-compatible: "+err.Error()+" — showing common OpenAI-style IDs.")
			return staticChatModelIDs("openai"), warns, nil
		}
		return filterLikelyChatModels(ids), warns, nil
	}
}

func openAICompatModelsURL() (string, error) {
	base := strings.TrimSpace(config.BaseURL())
	if base == "" {
		return "https://api.openai.com/v1/models", nil
	}
	return strings.TrimRight(base, "/") + "/models", nil
}

func fetchOllamaModelNames(ctx context.Context) ([]string, error) {
	v1 := strings.TrimRight(config.OllamaChatBase(), "/")
	root := strings.TrimSuffix(v1, "/v1")
	u := strings.TrimRight(root, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClientChatModel().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range parsed.Models {
		if n := strings.TrimSpace(m.Name); n != "" {
			out = append(out, n)
		}
	}
	return out, nil
}

func fetchGeminiGenerateContentModels(ctx context.Context, apiKey string) ([]string, error) {
	u := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClientChatModel().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	var parsed struct {
		Models []struct {
			Name                         string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range parsed.Models {
		ok := false
		for _, met := range m.SupportedGenerationMethods {
			if met == "generateContent" {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		id := strings.TrimSpace(m.Name)
		id = strings.TrimPrefix(id, "models/")
		if id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

func fetchOpenAICompatModelsList(ctx context.Context, listURL, bearer string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := httpClientChatModel().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var out []string
	for _, d := range parsed.Data {
		if id := strings.TrimSpace(d.ID); id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

func filterLikelyChatModels(ids []string) []string {
	if len(ids) <= 80 {
		return ids
	}
	var out []string
	for _, id := range ids {
		lo := strings.ToLower(id)
		if strings.Contains(lo, "embedding") || strings.Contains(lo, "moderation") {
			continue
		}
		if strings.Contains(lo, "tts") || strings.Contains(lo, "whisper") || strings.Contains(lo, "dall-e") || strings.Contains(lo, "audio") {
			continue
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return ids
	}
	return out
}

func staticChatModelIDs(provider string) []string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "ollama":
		return []string{
			"llama3.2", "llama3.1", "llama3", "mistral", "codellama", "phi3", "qwen2.5", "deepseek-r1",
		}
	case "gemini":
		return []string{
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-1.5-flash",
			"gemini-1.5-flash-8b",
			"gemini-1.5-pro",
			"gemini-2.5-flash-preview-05-20",
		}
	case "github":
		return []string{
			"gpt-4o", "gpt-4o-mini", "o1", "o1-mini", "o3-mini",
			"meta-llama-3.1-70b-instruct", "meta-llama-3.1-8b-instruct",
		}
	case "openrouter":
		return []string{
			"openai/gpt-4o", "openai/gpt-4o-mini", "anthropic/claude-3.5-sonnet",
			"google/gemini-2.0-flash-001", "meta-llama/llama-3.3-70b-instruct",
		}
	default:
		return []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo",
			"o1", "o1-mini", "o3-mini",
		}
	}
}

func dedupeSorted(sorted []string) []string {
	if len(sorted) == 0 {
		return sorted
	}
	out := sorted[:0]
	prev := ""
	for _, s := range sorted {
		if s == prev {
			continue
		}
		out = append(out, s)
		prev = s
	}
	return out
}

func httpClientChatModel() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}
