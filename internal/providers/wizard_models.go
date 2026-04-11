package providers

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"
)

// WizardDefaultBaseModels returns model IDs for the provider setup wizard when the user
// chose the official default API base. It uses environment credentials only (not viper),
// so it works before the wizard applies session config.
func WizardDefaultBaseModels(ctx context.Context, provider string) []string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	switch provider {
	case "openai":
		k := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		if k == "" {
			return nil
		}
		ids, err := FetchOpenAICompatModelsList(ctx, "https://api.openai.com/v1/models", k)
		if err != nil || len(ids) == 0 {
			return nil
		}
		sort.Strings(ids)
		return filterLikelyChatModels(dedupeSorted(ids))
	case "gemini":
		k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
		if k == "" {
			k = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
		}
		if k == "" {
			return nil
		}
		ids, err := FetchGeminiGenerateContentModels(ctx, k)
		if err != nil || len(ids) == 0 {
			return nil
		}
		sort.Strings(ids)
		return dedupeSorted(ids)
	case "github":
		ids := append([]string(nil), StaticChatModelIDs("github")...)
		sort.Strings(ids)
		return ids
	case "openrouter":
		k := strings.TrimSpace(os.Getenv("OPENROUTER_KEY"))
		if k == "" {
			k = strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
		}
		if k == "" {
			return nil
		}
		filter := strings.TrimSpace(strings.ToLower(os.Getenv("OPENROUTER_PROVIDER")))
		ids, err := FetchOpenRouterChatModelIDs(ctx, k, filter)
		if err != nil || len(ids) == 0 {
			return nil
		}
		sort.Strings(ids)
		return dedupeSorted(ids)
	default:
		return nil
	}
}

// WizardGitHubModelsAtBase lists model IDs from the GitHub Models OpenAI-compatible API at
// baseURL (non-empty), using GITHUB_TOKEN or GITHUB_PAT from the environment only.
// Returns nil on missing token, request error, or empty response (caller falls back to manual model entry).
func WizardGitHubModelsAtBase(ctx context.Context, baseURL string) []string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		return nil
	}
	tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if tok == "" {
		tok = strings.TrimSpace(os.Getenv("GITHUB_PAT"))
	}
	if tok == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	listURL := strings.TrimRight(base, "/") + "/models"
	ids, err := FetchOpenAICompatModelsList(ctx, listURL, tok)
	if err != nil || len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	return dedupeSorted(ids)
}
