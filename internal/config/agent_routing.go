package config

import (
	"strings"

	"github.com/spf13/viper"
)

// TaskAgentModel returns a non-empty model id to use for Task sub-agents, or empty to use the main client model.
// YAML: agent_routing.task_model — env: OPENCLAUDE_AGENT_TASK_MODEL (via viper).
func TaskAgentModel() string {
	return strings.TrimSpace(viper.GetString("agent_routing.task_model"))
}
