package installer

import "testing"

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		raw         string
		owner, repo string
		ref, sub    string
		ok          bool
	}{
		{"https://github.com/foo/bar", "foo", "bar", "", "", true},
		{"github.com/foo/bar/tree/main/src/x", "foo", "bar", "main", "src/x", true},
		{"https://github.com/foo/bar/blob/v1/README.md", "foo", "bar", "v1", "README.md", true},
		{"https://example.com/foo/bar", "", "", "", "", false},
	}
	for _, tc := range tests {
		o, r, ref, sub, ok := ParseGitHubURL(tc.raw)
		if ok != tc.ok || o != tc.owner || r != tc.repo || ref != tc.ref || sub != tc.sub {
			t.Fatalf("ParseGitHubURL(%q) = (%q,%q,%q,%q,%v) want (%q,%q,%q,%q,%v)",
				tc.raw, o, r, ref, sub, ok, tc.owner, tc.repo, tc.ref, tc.sub, tc.ok)
		}
	}
}
