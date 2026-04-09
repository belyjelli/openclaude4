package tools

// ghReadOnlySubcommands maps "sub sub2" (two tokens after optional global --repo/-R) to allowed flags.
// Derived from OpenClaude v3 src/utils/shell/readOnlyCommandValidation.ts GH_READ_ONLY_COMMANDS.
var ghReadOnlySubcommands = map[string]map[string]ghFlagKind{
	"pr view": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--json": ghFlagString, "--comments": ghFlagNone},
	),
	"pr list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--state": ghFlagString, "-s": ghFlagString, "--author": ghFlagString, "--assignee": ghFlagString,
			"--label": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber, "--base": ghFlagString,
			"--head": ghFlagString, "--search": ghFlagString, "--json": ghFlagString, "--draft": ghFlagNone,
			"--app": ghFlagString,
		},
	),
	"pr diff": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--color": ghFlagString, "--name-only": ghFlagNone, "--patch": ghFlagNone},
	),
	"pr checks": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--watch": ghFlagNone, "--required": ghFlagNone, "--fail-fast": ghFlagNone,
			"--json": ghFlagString, "--interval": ghFlagNumber,
		},
	),
	"issue view": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--json": ghFlagString, "--comments": ghFlagNone},
	),
	"issue list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--state": ghFlagString, "-s": ghFlagString, "--assignee": ghFlagString, "--author": ghFlagString,
			"--label": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber, "--milestone": ghFlagString,
			"--search": ghFlagString, "--json": ghFlagString, "--app": ghFlagString,
		},
	),
	"repo view": {
		"--json": ghFlagString,
	},
	"run list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--branch": ghFlagString, "-b": ghFlagString, "--status": ghFlagString, "-s": ghFlagString,
			"--workflow": ghFlagString, "-w": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber,
			"--json": ghFlagString, "--event": ghFlagString, "-e": ghFlagString, "--user": ghFlagString,
			"-u": ghFlagString, "--created": ghFlagString, "--commit": ghFlagString, "-c": ghFlagString,
		},
	),
	"run view": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--log": ghFlagNone, "--log-failed": ghFlagNone, "--exit-status": ghFlagNone,
			"--verbose": ghFlagNone, "-v": ghFlagNone, "--json": ghFlagString,
			"--job": ghFlagString, "-j": ghFlagString, "--attempt": ghFlagNumber, "-a": ghFlagNumber,
		},
	),
	"auth status": {
		"--active": ghFlagNone, "-a": ghFlagNone, "--hostname": ghFlagString, "-h": ghFlagString,
		"--json": ghFlagString,
	},
	"pr status": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--conflict-status": ghFlagNone, "-c": ghFlagNone, "--json": ghFlagString},
	),
	"issue status": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--json": ghFlagString},
	),
	"release list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--exclude-drafts": ghFlagNone, "--exclude-pre-releases": ghFlagNone, "--json": ghFlagString,
			"--limit": ghFlagNumber, "-L": ghFlagNumber, "--order": ghFlagString, "-O": ghFlagString,
		},
	),
	"release view": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{"--json": ghFlagString},
	),
	"workflow list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--all": ghFlagNone, "-a": ghFlagNone, "--json": ghFlagString,
			"--limit": ghFlagNumber, "-L": ghFlagNumber,
		},
	),
	"workflow view": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--ref": ghFlagString, "-r": ghFlagString, "--yaml": ghFlagNone, "-y": ghFlagNone,
		},
	),
	"label list": joinGhFlags(
		map[string]ghFlagKind{"--repo": ghFlagString, "-R": ghFlagString},
		map[string]ghFlagKind{
			"--json": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber,
			"--order": ghFlagString, "-O": ghFlagString, "--search": ghFlagString, "-S": ghFlagString,
			"--sort": ghFlagString,
		},
	),
	"search repos": ghSearchReposFlags(),
	"search issues": ghSearchIssuesFlags(),
	"search prs":    ghSearchPRsFlags(),
	"search commits": joinGhFlags(
		map[string]ghFlagKind{
			"--author": ghFlagString, "--author-date": ghFlagString, "--author-email": ghFlagString,
			"--author-name": ghFlagString, "--committer": ghFlagString, "--committer-date": ghFlagString,
			"--committer-email": ghFlagString, "--committer-name": ghFlagString, "--hash": ghFlagString,
			"--json": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber, "--merge": ghFlagNone,
			"--order": ghFlagString, "--owner": ghFlagString, "--parent": ghFlagString,
			"--repo": ghFlagString, "-R": ghFlagString, "--sort": ghFlagString, "--tree": ghFlagString,
			"--visibility": ghFlagString,
		},
	),
	"search code": {
		"--extension": ghFlagString, "--filename": ghFlagString, "--json": ghFlagString,
		"--language": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber,
		"--match": ghFlagString, "--owner": ghFlagString, "--repo": ghFlagString, "-R": ghFlagString,
		"--size": ghFlagString,
	},
}

func joinGhFlags(parts ...map[string]ghFlagKind) map[string]ghFlagKind {
	out := make(map[string]ghFlagKind, 48)
	for _, p := range parts {
		for k, v := range p {
			out[k] = v
		}
	}
	return out
}

func ghSearchReposFlags() map[string]ghFlagKind {
	return map[string]ghFlagKind{
		"--archived": ghFlagNone, "--created": ghFlagString, "--followers": ghFlagString,
		"--forks": ghFlagString, "--good-first-issues": ghFlagString, "--help-wanted-issues": ghFlagString,
		"--include-forks": ghFlagString, "--json": ghFlagString, "--language": ghFlagString,
		"--license": ghFlagString, "--limit": ghFlagNumber, "-L": ghFlagNumber, "--match": ghFlagString,
		"--number-topics": ghFlagString, "--order": ghFlagString, "--owner": ghFlagString,
		"--size": ghFlagString, "--sort": ghFlagString, "--stars": ghFlagString, "--topic": ghFlagString,
		"--updated": ghFlagString, "--visibility": ghFlagString,
	}
}

func ghSearchIssuesFlags() map[string]ghFlagKind {
	return map[string]ghFlagKind{
		"--app": ghFlagString, "--assignee": ghFlagString, "--author": ghFlagString, "--closed": ghFlagString,
		"--commenter": ghFlagString, "--comments": ghFlagString, "--created": ghFlagString,
		"--include-prs": ghFlagNone, "--interactions": ghFlagString, "--involves": ghFlagString,
		"--json": ghFlagString, "--label": ghFlagString, "--language": ghFlagString,
		"--limit": ghFlagNumber, "-L": ghFlagNumber, "--locked": ghFlagNone, "--match": ghFlagString,
		"--mentions": ghFlagString, "--milestone": ghFlagString, "--no-assignee": ghFlagNone,
		"--no-label": ghFlagNone, "--no-milestone": ghFlagNone, "--no-project": ghFlagNone,
		"--order": ghFlagString, "--owner": ghFlagString, "--project": ghFlagString,
		"--reactions": ghFlagString, "--repo": ghFlagString, "-R": ghFlagString, "--sort": ghFlagString,
		"--state": ghFlagString, "--team-mentions": ghFlagString, "--updated": ghFlagString,
		"--visibility": ghFlagString,
	}
}

func ghSearchPRsFlags() map[string]ghFlagKind {
	m := ghSearchIssuesFlags()
	extra := map[string]ghFlagKind{
		"--base": ghFlagString, "-B": ghFlagString, "--checks": ghFlagString, "--draft": ghFlagNone,
		"--head": ghFlagString, "-H": ghFlagString, "--merged": ghFlagNone, "--merged-at": ghFlagString,
		"--review": ghFlagString, "--review-requested": ghFlagString, "--reviewed-by": ghFlagString,
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}
