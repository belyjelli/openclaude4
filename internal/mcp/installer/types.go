package installer

// InstallRequest is the user-facing smart-install input.
type InstallRequest struct {
	URL           string
	SuggestedName string
	DryRun        bool
}

// RepoMetadata is minimal GitHub repo information for detectors.
type RepoMetadata struct {
	Owner         string
	Repo          string
	Ref           string
	SubPath       string // directory prefix inside repo (no leading slash)
	Description   string
	Language      string
	DefaultBranch string
	Topics        []string
}

// Candidate is one suggested MCP server configuration from a detector.
type Candidate struct {
	Name         string
	Transport    string
	Command      []string
	Env          map[string]string
	Approval     string
	ExtraArgs    []string
	Confidence   float64
	Reason       string
	DetectedFrom string
}
