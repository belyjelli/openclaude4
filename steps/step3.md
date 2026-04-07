**MCP Tool Support Expansion – Detailed Design (for openclaude-go)**

This document outlines a **complete, production-grade design** for adding full native support for the **Model Context Protocol (MCP)** to the Golang rewrite of OpenClaude. MCP is the open standard (launched by Anthropic in Nov 2024) that lets AI agents securely discover, connect to, and use external tools, data sources, and workflows — essentially a “USB-C port for AI”.

Because the current TypeScript OpenClaude and the belyjelli/openclaude4 design docs contain **zero MCP implementation**, this expansion positions your Go binary as one of the first fully MCP-native coding agents (alongside Claude Desktop/Code, Cursor, etc.).

### 1. Goals & Scope
- **Full parity** with official MCP clients (Claude Desktop, Cursor, Windsurf, etc.).
- **Zero extra dependencies** for core use (pure Go, no Node/Python runtimes).
- **Secure by default**: user consent for every dangerous action, per-server approval policies.
- **Dynamic discovery**: tools appear automatically when an MCP server is connected.
- **Multi-server support**: connect 1–N local + remote MCP servers simultaneously.
- **Transport agnostic**: support official transports (stdio, SSE, Streamable HTTP).
- **Seamless integration** into the existing agent/tool-calling loop (Phase 1).
- **Extensible**: easy to add new MCP features (resources, prompts, notifications) later.

**Non-goals (Phase 1 MCP)**: Becoming an MCP *server* (we can add that in v4.2). Only client-side consumption.

### 2. High-Level Architecture

```
internal/mcp/
├── client.go              # Core MCP client (connection + lifecycle)
├── transport/             # stdio, sse, http-streamable
├── protocol/              # JSON-RPC messages, schemas
├── discovery.go           # Auto-discovery & server registry
├── tool_adapter.go        # MCP tools → internal/tools.Tool interface
├── config.go              # MCP server profiles
├── manager.go             # Lifecycle + multi-server orchestration
└── security/              # Approval policies, sandboxing
```

The **MCP Manager** sits alongside the existing `tools.Registry` and `providers`. It acts as a **dynamic tool provider** that feeds discovered tools into the agent loop exactly like static tools (FileRead, Bash, etc.).

### 3. Configuration & Server Management

**Config file** (merged with existing `settings.json` and profiles):
```json
{
  "mcp": {
    "servers": [
      {
        "name": "github",
        "url": "stdio://github-mcp-server",
        "transport": "stdio",
        "command": ["npx", "@modelcontextprotocol/server-github"],
        "env": { "GITHUB_TOKEN": "..." },
        "autoConnect": true,
        "approvalPolicy": "always" | "ask" | "never",
        "deferLoading": true
      },
      {
        "name": "postgres-local",
        "url": "http://localhost:8080/sse",
        "transport": "sse",
        "autoConnect": true
      }
    ]
  }
}
```

**CLI commands** (added in Phase 1 MCP):
- `/mcp list` – show connected servers + tool count
- `/mcp connect <name|url>` – connect a new server
- `/mcp disconnect <name>`
- `/mcp refresh` – force tool re-discovery

Servers can also be passed via `--mcp-server stdio://...` flag or env var.

### 4. Connection & Lifecycle

**MCP Client States**:
1. **Disconnected**
2. **Connecting** (spawn process or HTTP connect)
3. **Initialized** (after `initialize` handshake)
4. **Ready** (tools/resources/prompts discovered)
5. **Error** / **Disconnected** (auto-reconnect with backoff)

**Handshake flow** (standard MCP JSON-RPC):
- Client → Server: `initialize` (client info, protocol version)
- Server → Client: `initialize` response (server capabilities)
- Client → Server: `tools/list`, `resources/list`, `prompts/list`
- Ongoing: `tools/call`, `resources/read`, notifications, ping/pong

**Transport implementations** (priority order):
1. **stdio** (most common for local servers) – spawn process, pipe JSON-RPC
2. **SSE** (Server-Sent Events) – for remote/lightweight servers
3. **Streamable HTTP** (future-proof)

All transports implement the same `Transport` interface internally.

### 5. Tool Discovery & Adaptation

When a server reaches “Ready” state:
- Call `tools/list` → receive array of tool definitions (name, description, inputSchema, etc.)
- For each tool, create a **thin adapter** that implements `internal/tools.Tool`:
  - `Name()` = MCP tool name (prefixed with server name, e.g. `github:search_repo`)
  - `Description()` = MCP description + server name
  - `Parameters()` = converted from JSON Schema
  - `IsDangerous()` = derived from server policy + tool metadata
  - `Execute()` = calls `tools/call` on the MCP server and returns result

**Tool registration**:
- Tools are **not** registered in the static registry.
- They live in a **dynamic MCPToolSet** per server.
- The agent loop queries both static tools + all active MCP toolsets.

**Defer loading** support (as per Anthropic best practices):
- Server can be connected but tools only loaded when first needed (saves tokens/context).

### 6. Integration into Agent/Tool-Calling Loop

No changes required to the core `internal/core/agent.go` loop from Phase 1.

The `provider.ChatWithTools()` call already receives the full list from:
```go
tools := append(staticTools, mcpManager.GetAllTools()...)
```

When the model emits a tool call:
- If the tool name contains `:` (e.g. `postgres:query_db`), route to the correct MCP server.
- Execute via `tools/call` RPC.
- Result is fed back exactly like any other tool (as a `tool` role message).

**Streaming & progress**:
- MCP servers can send `notifications` during long operations.
- Forward these to the TUI as live status updates (e.g. “Cloning repo…”).

### 7. Security & Approvals (Critical)

**Per-server approval policies** (configurable):
- `always` – auto-approve all calls
- `ask` – TUI confirmation dialog before every `tools/call`
- `never` – block dangerous tools (configurable blacklist)

**Global safeguards**:
- All MCP tool executions respect the same sandbox cwd limits as Bash/FileWrite.
- Sensitive data (tokens, keys) never logged in plaintext.
- Connection isolation: each server runs in its own goroutine with separate context cancellation.

**User consent UI** (Bubble Tea):
- One-time “Connect to XYZ MCP server?” dialog on first connect.
- Per-action confirmation for high-risk tools (write, delete, network, etc.).

### 8. Resource & Prompt Support (Phase 1.5)

After basic tools work, add:
- **Resources**: `resources/list` + `resources/read` → expose as read-only `MCPResourceRead` tool.
- **Prompts**: `prompts/list` + `prompts/get` → expose as special `/mcp prompt <name>` slash command.

These are treated as **first-class citizens** alongside tools.

### 9. Error Handling & Resilience

- Graceful disconnect/reconnect on process crash or network drop.
- Detailed error messages surfaced to user (e.g. “MCP server ‘github’ returned: rate limit exceeded”).
- Logging: optional `--mcp-debug` flag dumps full JSON-RPC traffic.
- Metrics: track tool calls per server, latency, token usage (if server reports it).

### 10. Testing & Validation Strategy

**Unit tests**:
- Mock JSON-RPC transport for protocol compliance.

**Integration tests**:
- Use official sample MCP servers (GitHub, Google Drive, Postgres from modelcontextprotocol/servers repo).
- End-to-end: connect → list tools → agent uses MCP tool in real coding task.

**Compliance**:
- Follow official spec: https://modelcontextprotocol.io/docs
- Test against real clients (Claude Desktop) to ensure interoperability.

### 11. Roadmap Integration

- **Phase 1 MCP** (this design): tools + basic resources + config + security
- **Phase 2 MCP**: prompts, notifications, hosted/remote servers, auto-discovery via mcp-hub
- **Phase 3 MCP**: Become an MCP *server* (expose OpenClaude tools to other agents)

This design makes openclaude-go **MCP-first** — the agent will automatically gain hundreds of community tools (databases, GitHub, Slack, Figma, Zapier, etc.) the moment the user adds a server.

Would you like me to:
- Expand this into a full `DESIGN.md` section with diagrams (text-based)?
- Add the exact config schema + CLI command reference?
- Or outline how this fits into the existing provider/tool registry interfaces?

Just let me know which part to deepen next! This will be a massive differentiator for the Go version.