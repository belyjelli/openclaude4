package installer

import (
	"context"
	"fmt"
)

// GenericFallback suggests a git clone when nothing else matches.
type GenericFallback struct{}

func (GenericFallback) Name() string              { return "GenericFallback" }
func (GenericFallback) ConfidenceWeight() float64 { return 0.1 }

func (GenericFallback) Detect(_ context.Context, _ map[string][]byte, meta *RepoMetadata) ([]*Candidate, error) {
	if meta == nil {
		return nil, nil
	}
	owner, repo := meta.Owner, meta.Repo
	if owner == "" || repo == "" {
		return nil, nil
	}
	url := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	cmd := []string{"git", "clone", url, repo}
	return []*Candidate{{
		Name:         sanitizeServerName(repo, repo),
		Transport:    "stdio",
		Command:      cmd,
		Approval:     "ask",
		Confidence:   5,
		Reason:       "Generic fallback: clone repository (inspect locally, then configure MCP manually)",
		DetectedFrom: "GenericFallback",
	}}, nil
}
