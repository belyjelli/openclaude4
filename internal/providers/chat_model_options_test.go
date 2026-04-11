package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestDedupeSorted(t *testing.T) {
	in := []string{"a", "a", "b", "c", "c"}
	sort.Strings(in)
	got := dedupeSorted(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestFilterLikelyChatModels(t *testing.T) {
	in := make([]string, 0, 85)
	for i := range 81 {
		in = append(in, fmt.Sprintf("chat-model-%d", i))
	}
	in = append(in, "text-embedding-3-small")
	got := filterLikelyChatModels(in)
	for _, id := range got {
		if id == "text-embedding-3-small" {
			t.Fatalf("embedding model should be filtered: %v", got)
		}
	}
	if len(got) >= len(in) {
		t.Fatalf("expected some filtering, got len %d", len(got))
	}
}

func TestFetchOpenAICompatModelsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o"},{"id":"gpt-4o-mini"}]}`))
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	got, err := FetchOpenAICompatModelsList(ctx, srv.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "gpt-4o" || got[1] != "gpt-4o-mini" {
		t.Fatalf("got %v", got)
	}
}
