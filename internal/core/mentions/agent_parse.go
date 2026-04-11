package mentions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
)

// v3: (^|\s)@"([\w:.@-]+) \(agent\)"
var quotedAgentRe = regexp.MustCompile(`(?:^|\s)@"([\w:.@-]+) \(agent\)"`)

// v3 legacy inner token after @
var agentInnerTokenRe = regexp.MustCompile(`^agent-[\w:.@-]+$`)

// agentSyntaxInner is true when inner is @"… (agent)" content or @agent-<type>, not e.g. ./agent-foo.go.
func agentSyntaxInner(inner string) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}
	if strings.HasSuffix(inner, " (agent)") {
		return true
	}
	return agentInnerTokenRe.MatchString(inner)
}

// ExtractAgentTypes returns distinct agent type ids from user text (order of first appearance).
func ExtractAgentTypes(userText string) ([]string, error) {
	seen := map[string]struct{}{}
	var order []string

	for _, m := range quotedAgentRe.FindAllStringSubmatch(userText, -1) {
		if len(m) < 2 {
			continue
		}
		t := strings.TrimSpace(m[1])
		if t == "" {
			continue
		}
		low := strings.ToLower(t)
		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		order = append(order, t)
	}

	ua := regexp.MustCompile(`(?:^|\s)@(agent-[\w:.@-]+)`)
	for _, m := range ua.FindAllStringSubmatch(userText, -1) {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if !agentInnerTokenRe.MatchString(raw) {
			continue
		}
		t := strings.TrimPrefix(raw, "agent-")
		if t == "" {
			continue
		}
		low := strings.ToLower(t)
		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		order = append(order, t)
	}

	return order, nil
}

const maxAgentInjectBytes = 32 * 1024

func buildAgentPrefix(userText string, byType map[string]config.AgentProfile) (prefix string, injectBytes int, err error) {
	if len(byType) == 0 {
		return "", 0, nil
	}
	types, err := ExtractAgentTypes(userText)
	if err != nil {
		return "", 0, err
	}
	if len(types) == 0 {
		return "", 0, nil
	}
	var b strings.Builder
	total := 0
	for _, t := range types {
		p, ok := byType[strings.ToLower(strings.TrimSpace(t))]
		if !ok {
			return "", 0, fmt.Errorf("unknown @agent type %q: add it under agents: in openclaude.yaml", t)
		}
		sec := "### Agent: " + p.Type + "\n" + strings.TrimSpace(p.Instructions) + "\n\n"
		n := len(sec)
		if total+n > maxAgentInjectBytes {
			return "", 0, fmt.Errorf("agent instruction blocks exceed max (%d bytes)", maxAgentInjectBytes)
		}
		total += n
		b.WriteString(sec)
	}
	return b.String(), total, nil
}
