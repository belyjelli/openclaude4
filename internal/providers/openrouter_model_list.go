package providers

import (
	"context"
	"strings"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
)

// FetchOpenRouterChatModelIDs lists model ids from the OpenRouter catalog using the official go-sdk
// (github.com/OpenRouterTeam/go-sdk). providerSlug filters to ids starting with "<slug>/"; empty = all chat-suitable models.
func FetchOpenRouterChatModelIDs(ctx context.Context, apiKey, providerSlug string) ([]string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, nil
	}
	slug := strings.Trim(strings.TrimSpace(strings.ToLower(providerSlug)), "/")
	client := openrouter.New(openrouter.WithSecurity(apiKey))
	res, err := client.Models.List(ctx, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	prefix := ""
	if slug != "" {
		prefix = slug + "/"
	}
	var out []string
	for i := range res.Data {
		m := &res.Data[i]
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(id), prefix) {
			continue
		}
		if skipOpenRouterModelForChat(id, m) {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func skipOpenRouterModelForChat(id string, m *components.Model) bool {
	lo := strings.ToLower(id)
	if strings.Contains(lo, "text-embedding") || strings.Contains(lo, "/embed") {
		return true
	}
	if strings.HasSuffix(lo, ":free-embed") || strings.Contains(lo, "embeddings") {
		return true
	}
	if m == nil {
		return false
	}
	mod := m.Architecture.Modality
	if mod == nil {
		return false
	}
	ms := strings.ToLower(strings.TrimSpace(*mod))
	return ms == "embedding" || ms == "embeddings"
}
