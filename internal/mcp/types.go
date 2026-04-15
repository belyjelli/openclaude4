package mcp

// ConnectionState is reserved for the lifecycle manager (MCP v2 roadmap).
type ConnectionState string

const (
	StatePending    ConnectionState = "pending"
	StateConnecting ConnectionState = "connecting"
	StateConnected  ConnectionState = "connected"
	StateFailed     ConnectionState = "failed"
	StateNeedsAuth  ConnectionState = "needs_auth"
	StateDisabled   ConnectionState = "disabled"
)

// ServerConfig is one MCP server after config resolution (v2 or legacy bridge).
type ServerConfig struct {
	Name       string
	Transport  string // stdio | http | sse | ws (only stdio is implemented)
	Command    []string
	Env        map[string]string
	Approval   string         // ask | always | never
	Policies   map[string]any // optional allow/deny from v2 YAML
	ConfigPath string         // set when loaded from a v2 file (for tooling)
}
