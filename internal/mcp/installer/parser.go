package installer

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Detector parses repo files into install candidates.
type Detector interface {
	Name() string
	Detect(ctx context.Context, files map[string][]byte, meta *RepoMetadata) ([]*Candidate, error)
	ConfidenceWeight() float64
}

// ParseGitHubRepo fetches lightweight repo files and runs detectors.
func ParseGitHubRepo(ctx context.Context, client *http.Client, req InstallRequest) (*RepoMetadata, []*Candidate, error) {
	if client == nil {
		client = NewHTTPClient()
	}
	owner, repo, ref, subPath, ok := ParseGitHubURL(req.URL)
	if !ok {
		return nil, nil, fmt.Errorf("not a supported GitHub repository URL")
	}
	meta := &RepoMetadata{Owner: owner, Repo: repo, SubPath: subPath}

	defaultBranch, desc, lang, topics, err := RepoAPIMetadata(ctx, client, owner, repo)
	if err != nil {
		return nil, nil, err
	}
	meta.Description = desc
	meta.Language = lang
	meta.Topics = append([]string(nil), topics...)
	meta.DefaultBranch = defaultBranch
	if ref == "" {
		ref = strings.TrimSpace(defaultBranch)
	}
	if ref == "" {
		ref = "main"
	}
	meta.Ref = ref

	files, err := fetchCommonFiles(ctx, client, meta)
	if err != nil {
		return meta, nil, err
	}

	var dets []Detector
	dets = append(dets, NPM{}, ReadmeCommands{}, GenericFallback{})

	var all []*Candidate
	maxConf := 0.0
	for _, d := range dets {
		got, err := d.Detect(ctx, files, meta)
		if err != nil {
			continue
		}
		for _, c := range got {
			if c == nil {
				continue
			}
			c.Confidence = c.Confidence * d.ConfidenceWeight()
			if c.Confidence > maxConf {
				maxConf = c.Confidence
			}
			all = append(all, c)
		}
	}

	// Drop generic fallback when a stronger signal exists.
	if maxConf >= 40 {
		var filtered []*Candidate
		for _, c := range all {
			if c.DetectedFrom == "GenericFallback" {
				continue
			}
			filtered = append(filtered, c)
		}
		all = filtered
	}

	all = dedupeCandidates(all)
	sort.Slice(all, func(i, j int) bool { return all[i].Confidence > all[j].Confidence })
	return meta, all, nil
}

func repoRel(subPath, name string) string {
	subPath = strings.Trim(strings.TrimPrefix(subPath, "/"), "/")
	if subPath == "" {
		return name
	}
	return subPath + "/" + name
}

func fetchCommonFiles(ctx context.Context, client *http.Client, meta *RepoMetadata) (map[string][]byte, error) {
	out := map[string][]byte{}
	names := []string{
		"README.md", "README.markdown", "readme.md",
		"package.json",
		"go.mod",
		"bun.lockb", "bunfig.toml",
	}
	for _, n := range names {
		rel := repoRel(meta.SubPath, n)
		b, err := FetchRaw(ctx, client, meta.Owner, meta.Repo, meta.Ref, rel)
		if err == nil && len(b) > 0 {
			out[n] = b
		}
	}
	return out, nil
}

func dedupeCandidates(in []*Candidate) []*Candidate {
	seen := map[string]struct{}{}
	var out []*Candidate
	for _, c := range in {
		if c == nil {
			continue
		}
		sig := strings.Join(c.Command, "\x00")
		if _, ok := seen[sig]; ok {
			continue
		}
		seen[sig] = struct{}{}
		out = append(out, c)
	}
	return out
}
