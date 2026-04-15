package installer

import (
	"context"
	"regexp"
	"strings"
)

// ReadmeCommands extracts shell commands from README lines (fallback detector).
type ReadmeCommands struct{}

func (ReadmeCommands) Name() string              { return "ReadmeCommandDetector" }
func (ReadmeCommands) ConfidenceWeight() float64 { return 0.5 }

var lineCmdPattern = regexp.MustCompile(`(?m)^\s*(npx|bunx|uvx)\s+(.+)$`)

func (ReadmeCommands) Detect(_ context.Context, files map[string][]byte, meta *RepoMetadata) ([]*Candidate, error) {
	if meta == nil {
		return nil, nil
	}
	var readme []byte
	for k, v := range files {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "readme") && strings.HasSuffix(lk, ".md") {
			readme = v
			break
		}
	}
	if len(readme) == 0 {
		return nil, nil
	}
	var out []*Candidate
	seen := map[string]struct{}{}
	for _, m := range lineCmdPattern.FindAllStringSubmatch(string(readme), -1) {
		if len(m) < 3 {
			continue
		}
		verb := strings.ToLower(strings.TrimSpace(m[1]))
		rest := strings.TrimSpace(m[2])
		if rest == "" {
			continue
		}
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		cmd := append([]string{verb}, parts...)
		sig := strings.Join(cmd, " ")
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}

		conf := 35.0
		line := strings.ToLower(rest)
		if strings.Contains(line, "mcp") || strings.Contains(line, "@modelcontextprotocol") {
			conf += 25
		}
		if verb == "bunx" && hasBunInstalled() {
			conf += 10
		}
		if conf > 95 {
			conf = 95
		}
		out = append(out, &Candidate{
			Name:         sanitizeServerName(meta.Repo, meta.Repo),
			Transport:    "stdio",
			Command:      cmd,
			Approval:     "ask",
			Confidence:   conf,
			Reason:       "README command line",
			DetectedFrom: "ReadmeCommandDetector",
		})
	}
	return out, nil
}
