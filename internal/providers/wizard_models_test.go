package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWizardGitHubModelsAtBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer gh-test" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o"},{"id":"gpt-4o-mini"}]}`))
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	t.Run("no token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GITHUB_PAT", "")
		if got := WizardGitHubModelsAtBase(ctx, srv.URL); got != nil {
			t.Fatalf("want nil, got %v", got)
		}
	})
	t.Run("with token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "gh-test")
		t.Setenv("GITHUB_PAT", "")
		got := WizardGitHubModelsAtBase(ctx, srv.URL)
		if len(got) != 2 || got[0] != "gpt-4o" || got[1] != "gpt-4o-mini" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("GITHUB_PAT fallback", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GITHUB_PAT", "gh-test")
		got := WizardGitHubModelsAtBase(ctx, srv.URL)
		if len(got) != 2 {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("empty base", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "gh-test")
		if got := WizardGitHubModelsAtBase(ctx, "  "); got != nil {
			t.Fatalf("want nil, got %v", got)
		}
	})
	t.Run("http error", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "wrong")
		if got := WizardGitHubModelsAtBase(ctx, srv.URL); got != nil {
			t.Fatalf("want nil, got %v", got)
		}
	})
}
