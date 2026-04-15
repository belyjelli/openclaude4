package v2

// Server is one MCP v2 server entry (on-disk shape; no import of parent mcp package).
type Server struct {
	Name       string
	Transport  string
	Command    []string
	Env        map[string]string
	Approval   string
	Policies   map[string]any
	ConfigPath string
}
