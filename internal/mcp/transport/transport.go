package transport

import "context"

// Transport connects an MCP server (roadmap: stdio, sse, ws, http).
type Transport interface {
	Name() string
	Connect(ctx context.Context) error
	Close() error
}
