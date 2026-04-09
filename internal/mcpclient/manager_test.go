package mcpclient

import (
	"context"
	"reflect"
	"testing"

	"github.com/gitlawb/openclaude4/internal/tools"
)

func TestConnectAndRegister_empty(t *testing.T) {
	reg := tools.NewRegistry()
	m := ConnectAndRegister(context.Background(), reg, nil, nil)
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
	m.Close()
}

func TestManager_Close_nil(t *testing.T) {
	var m *Manager
	m.Close()
}

func TestManager_ResourceSuggestCandidates_nil(t *testing.T) {
	var m *Manager
	if got := m.ResourceSuggestCandidates(""); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestResourceSuggestCandidates_table(t *testing.T) {
	base := &Manager{
		Servers: []ServerTools{
			{
				Name: "srv",
				Resources: []MCPResource{
					{Server: "srv", URI: "scheme://foo/bar", Name: "fooDoc", Title: "Foo Title"},
					{Server: "srv", URI: "other://x", Name: "nomatch", Title: "Another"},
					{Server: "srv", URI: "z://last", Name: "prefixStem", Title: ""},
				},
			},
		},
	}

	tests := []struct {
		query string
		want  []string // URIs in order
	}{
		{"", []string{"other://x", "scheme://foo/bar", "z://last"}},
		{"scheme", []string{"scheme://foo/bar"}},
		{"SCHEME", []string{"scheme://foo/bar"}},
		{"foo", []string{"scheme://foo/bar"}}, // prefix on URI
		{"foodoc", []string{"scheme://foo/bar"}},
		{"foo title", []string{"scheme://foo/bar"}},
		{"prefix", []string{"z://last"}},
		{"nomatch", []string{"other://x"}},
		{"zzz", nil},
	}

	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			got := base.ResourceSuggestCandidates(tc.query)
			var uris []string
			for _, r := range got {
				uris = append(uris, r.URI)
			}
			if !reflect.DeepEqual(uris, tc.want) {
				t.Fatalf("query %q: got %v want %v", tc.query, uris, tc.want)
			}
		})
	}
}

func TestResourceSuggestCandidates_dedupeByURI(t *testing.T) {
	m := &Manager{
		Servers: []ServerTools{
			{Name: "s1", Resources: []MCPResource{{Server: "s1", URI: "same://u", Name: "a"}}},
			{Name: "s2", Resources: []MCPResource{{Server: "s2", URI: "same://u", Name: "b"}}},
		},
	}
	got := m.ResourceSuggestCandidates("")
	if len(got) != 1 {
		t.Fatalf("len=%d want 1 (dedupe by URI)", len(got))
	}
	if got[0].URI != "same://u" {
		t.Fatalf("URI=%q", got[0].URI)
	}
}
