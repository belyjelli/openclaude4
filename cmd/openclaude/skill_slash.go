package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gitlawb/openclaude4/internal/core"
	"github.com/gitlawb/openclaude4/internal/hooks"
	"github.com/gitlawb/openclaude4/internal/sandbox"
	"github.com/gitlawb/openclaude4/internal/skills"
	sdk "github.com/sashabaranov/go-openai"
)

type sandboxShellRunner struct{}

func (sandboxShellRunner) RunShell(ctx context.Context, command, cwd string) (string, error) {
	return sandbox.RunShell(ctx, command, cwd)
}

// handleUserSkillSlash runs v3-style /skill invocation: expand body, optional fork, or SlashSubmitUser for inline.
func handleUserSkillSlash(ctx context.Context, line string, fields []string, e skills.Entry, st chatState, out io.Writer) error {
	if e.DisableModelInvocation {
		_, _ = fmt.Fprintf(out, "Skill %q can only be invoked by the model (disable_model_invocation).\n", e.Name)
		return nil
	}
	rawTail := ""
	if len(fields) > 0 {
		rawTail = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), fields[0]))
	}
	sid := ""
	if st.persist != nil {
		sid = st.persist.store.ID
	}
	var runner skills.PromptShellRunner = sandboxShellRunner{}
	expanded, err := skills.FullExpand(ctx, e, rawTail, sid, runner)
	if err != nil {
		return err
	}
	if len(e.Hooks) > 0 && sid != "" {
		_ = hooks.Default().RegisterFromSkill(sid, e.Dir, e.Hooks)
	}
	if strings.EqualFold(e.Context, "fork") {
		ag := st.resolveAgent()
		if ag == nil {
			return fmt.Errorf("forked skill requires an active agent (unavailable here)")
		}
		max := e.MaxForkIterations
		allow := skills.NormalizeAllowedToolList(e.AllowedTools)
		summary, err := core.RunSkillForked(ctx, ag, allow, expanded, max)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "(forked skill /%s completed)\n", e.Name)
		if st.messages != nil {
			*st.messages = append(*st.messages,
				sdk.ChatCompletionMessage{
					Role:    sdk.ChatMessageRoleUser,
					Content: fmt.Sprintf("(forked skill /%s)", e.Name),
				},
				sdk.ChatCompletionMessage{
					Role:    sdk.ChatMessageRoleAssistant,
					Content: summary,
				},
			)
			if st.persist != nil {
				_ = st.persist.Save()
			}
		}
		return nil
	}
	allowNorm := skills.NormalizeAllowedToolList(e.AllowedTools)
	su := core.SlashSubmitUser{UserText: expanded}
	if len(allowNorm) > 0 {
		su.AllowTools = allowNorm
	}
	return su
}
