package skills

import (
	"context"
	"regexp"
	"strings"
)

// PromptShellRunner runs one shell command (e.g. sandbox.RunShell) for SKILL.md embedded shell.
type PromptShellRunner interface {
	RunShell(ctx context.Context, command, cwd string) (stdout string, err error)
}

var (
	blockShellPattern  = regexp.MustCompile("(?s)```!\\s*\n?(.*?)```")
	inlineShellPattern = regexp.MustCompile(`(?:^|\s)!\x60([^\x60]+)\x60`)
)

// ExecuteShellCommandsInPrompt expands ```! ... ``` and !`cmd` patterns after argument substitution.
// If runner is nil or allowLocal is false, returns text unchanged (v3: skip for non-local skills).
func ExecuteShellCommandsInPrompt(ctx context.Context, text string, cwd string, runner PromptShellRunner, allowLocal bool) (string, error) {
	if runner == nil || !allowLocal {
		return text, nil
	}
	var err error
	text, err = replaceAllShellBlocks(ctx, text, cwd, runner)
	if err != nil {
		return "", err
	}
	if strings.Contains(text, "!\x60") {
		text, err = replaceInlineShell(ctx, text, cwd, runner)
		if err != nil {
			return "", err
		}
	}
	return text, nil
}

func replaceAllShellBlocks(ctx context.Context, text, cwd string, runner PromptShellRunner) (string, error) {
	for {
		loc := blockShellPattern.FindStringSubmatchIndex(text)
		if loc == nil {
			return text, nil
		}
		fullStart, fullEnd := loc[0], loc[1]
		cmdStart, cmdEnd := loc[2], loc[3]
		cmd := strings.TrimSpace(text[cmdStart:cmdEnd])
		if cmd == "" {
			text = text[:fullStart] + text[fullEnd:]
			continue
		}
		stdout, err := runner.RunShell(ctx, cmd, cwd)
		if err != nil {
			return "", err
		}
		text = text[:fullStart] + strings.TrimSpace(stdout) + text[fullEnd:]
	}
}

func replaceInlineShell(ctx context.Context, text, cwd string, runner PromptShellRunner) (string, error) {
	for {
		loc := inlineShellPattern.FindStringSubmatchIndex(text)
		if loc == nil {
			return text, nil
		}
		fullStart, fullEnd := loc[0], loc[1]
		cmdStart, cmdEnd := loc[2], loc[3]
		cmd := strings.TrimSpace(text[cmdStart:cmdEnd])
		if cmd == "" {
			text = text[:fullStart] + text[fullEnd:]
			continue
		}
		stdout, err := runner.RunShell(ctx, cmd, cwd)
		if err != nil {
			return "", err
		}
		repl := strings.TrimSpace(stdout)
		matched := text[fullStart:fullEnd]
		prefix := ""
		if len(matched) > 0 && (matched[0] == ' ' || matched[0] == '\t' || matched[0] == '\n') {
			prefix = string(matched[0])
		}
		text = text[:fullStart] + prefix + repl + text[fullEnd:]
	}
}
