package skills

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
)

// ExpandSkillBody applies base-dir prefix, $ARGUMENTS-style substitution, and
// ${CLAUDE_SKILL_DIR} / ${CLAUDE_SESSION_ID} replacements (v3-aligned).
func ExpandSkillBody(entry Entry, rawArgsTail string, sessionID string) string {
	body := entry.Body
	if entry.Dir != "" {
		body = "Base directory for this skill: " + entry.Dir + "\n\n" + body
	}
	out := SubstituteArguments(body, rawArgsTail, true, entry.ArgumentNames)
	skillDir := entry.Dir
	if runtime.GOOS == "windows" {
		skillDir = filepath.ToSlash(skillDir)
	}
	out = strings.ReplaceAll(out, "${CLAUDE_SKILL_DIR}", skillDir)
	out = strings.ReplaceAll(out, "${CLAUDE_SESSION_ID}", sessionID)
	return out
}

// FullExpand runs argument + path substitution then optional embedded shell in SKILL.md.
func FullExpand(ctx context.Context, entry Entry, rawArgsTail, sessionID string, runner PromptShellRunner) (string, error) {
	s := ExpandSkillBody(entry, rawArgsTail, sessionID)
	cwd := entry.Dir
	if cwd == "" {
		cwd = "."
	}
	return ExecuteShellCommandsInPrompt(ctx, s, cwd, runner, entry.SourceLocal)
}
