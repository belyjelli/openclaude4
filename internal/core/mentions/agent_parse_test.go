package mentions

import (
	"strings"
	"testing"

	"github.com/gitlawb/openclaude4/internal/config"
)

func TestExtractAgentTypes_quotedAndLegacy(t *testing.T) {
	s := `hello @"code-reviewer (agent)" and @agent-lint:strict x`
	got, err := ExtractAgentTypes(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "code-reviewer" || got[1] != "lint:strict" {
		t.Fatalf("got %#v", got)
	}
}

func TestBuildAgentPrefix_unknown(t *testing.T) {
	by := map[string]config.AgentProfile{
		"a": {Type: "a", Instructions: "do a"},
	}
	_, _, err := buildAgentPrefix("@agent-b", by)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("got %v", err)
	}
}

func TestBuildAgentPrefix_ok(t *testing.T) {
	by := map[string]config.AgentProfile{
		"rev": {Type: "rev", Instructions: "review code"},
	}
	p, n, err := buildAgentPrefix("fix @agent-rev", by)
	if err != nil || n == 0 || !strings.Contains(p, "### Agent: rev") || !strings.Contains(p, "review code") {
		t.Fatalf("p=%q n=%d err=%v", p, n, err)
	}
}
