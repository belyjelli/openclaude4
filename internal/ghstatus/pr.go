package ghstatus

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PrReviewState matches OpenClaude v3 ghPrStatus deriveReviewState outcomes.
type PrReviewState string

const (
	PrApproved          PrReviewState = "approved"
	PrPending           PrReviewState = "pending"
	PrChangesRequested  PrReviewState = "changes_requested"
	PrDraft             PrReviewState = "draft"
	PrMerged            PrReviewState = "merged"
	PrClosed            PrReviewState = "closed"
)

// OpenPRStatus is a non-closed, non-merged open PR for the current branch (when applicable).
type OpenPRStatus struct {
	Number       int
	URL          string
	ReviewState  PrReviewState
	HeadRefName  string
	State        string // OPEN, MERGED, CLOSED from API
}

func deriveReviewState(isDraft bool, reviewDecision string) PrReviewState {
	if isDraft {
		return PrDraft
	}
	switch reviewDecision {
	case "APPROVED":
		return PrApproved
	case "CHANGES_REQUESTED":
		return PrChangesRequested
	default:
		return PrPending
	}
}

// FetchOpenPRForCWD runs `gh pr view` like OpenClaude v3 fetchPrStatus: only when in a git repo,
// not on the default branch, and only for an open PR whose head is not the repo default branch name.
func FetchOpenPRForCWD(ctx context.Context, cwd string) (*OpenPRStatus, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, nil
	}
	if cwd == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !gitIsRepo(ctx, cwd) {
		return nil, nil
	}
	branch, err := gitCurrentBranch(ctx, cwd)
	if err != nil || branch == "" {
		return nil, nil
	}
	defaultBranch, err := gitOriginDefaultBranch(ctx, cwd)
	if err != nil {
		defaultBranch = ""
	}
	if defaultBranch != "" && branch == defaultBranch {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json",
		"number,url,reviewDecision,isDraft,headRefName,state")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	var data struct {
		Number           int    `json:"number"`
		URL              string `json:"url"`
		ReviewDecision   string `json:"reviewDecision"`
		IsDraft          bool   `json:"isDraft"`
		HeadRefName      string `json:"headRefName"`
		State            string `json:"state"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}

	if data.HeadRefName == defaultBranch || data.HeadRefName == "main" || data.HeadRefName == "master" {
		return nil, nil
	}
	if data.State == "MERGED" || data.State == "CLOSED" {
		return nil, nil
	}

	st := deriveReviewState(data.IsDraft, data.ReviewDecision)
	return &OpenPRStatus{
		Number:      data.Number,
		URL:         data.URL,
		ReviewState: st,
		HeadRefName: data.HeadRefName,
		State:       data.State,
	}, nil
}

// FormatShort returns a compact status-bar fragment, or empty when nil.
func (s *OpenPRStatus) FormatShort() string {
	if s == nil {
		return ""
	}
	rs := string(s.ReviewState)
	switch s.ReviewState {
	case PrApproved:
		rs = "approved"
	case PrPending:
		rs = "pending"
	case PrChangesRequested:
		rs = "changes"
	case PrDraft:
		rs = "draft"
	}
	return "PR #" + strconv.Itoa(s.Number) + " · " + rs
}
