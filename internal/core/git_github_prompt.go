package core

// GitAndGitHubWorkflowInstructions aligns agent behavior with OpenClaude v3: git for local
// repo work; GitHub CLI (gh) for issues, PRs, checks, releases, and structured API access.
const GitAndGitHubWorkflowInstructions = `

# Git and GitHub
- **Local git:** Use git for status, diff, log, branch, add, commit, and push. Do not change git config unless the user asks. Avoid destructive operations (force-push to main/master, reset --hard, checkout/restore/clean that discards work, branch -D) unless the user explicitly requests them. Do not skip hooks (--no-verify, --no-gpg-sign) unless the user asks. If a pre-commit hook fails, fix the issue and create a new commit rather than amending a prior one. Prefer staging explicit paths instead of git add -A or git add dot. Only create commits when the user asked; never commit files that likely contain secrets.
- **GitHub:** Use the GitHub CLI (gh) via Bash for issues, pull requests, CI checks, releases, and other GitHub API data. For github.com URLs, prefer gh (for example: gh pr view, gh issue view, gh api) over WebFetch when you need private repos, structured JSON, or PR/issue details. The user should have gh installed and authenticated (gh auth login).
- **Pull requests:** Inspect the branch with git (status, diff, log; compare against the base branch). Push if needed, then run gh pr create with a concise title and a detailed body; use a shell HEREDOC for multi-line bodies. Return the PR URL when finished.
`
