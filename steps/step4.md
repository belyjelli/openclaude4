**MCP Security Policies & Guidelines for openclaude-go**  
(Designed to be safer and more explicit than the original Gitlawb/openclaude while aligning with its spirit and official MCP best practices)

The original OpenClaude (Gitlawb/openclaude) has **minimal built-in security enforcement** for tools:
- Dangerous operations (Bash, FileWrite, FileEdit) run directly in the host environment with the user's privileges.
- No documented sandboxing, command blacklisting, or mandatory per-action confirmation in the main CLI (only partial prompts appear in the gRPC headless mode via `action_required` events).
- API keys and provider credentials are stored in plaintext in `~/.claude/settings.json` (with a clear warning not to commit the file).
- Tool execution follows the model’s requests with little friction, relying on user oversight during interactive sessions.
- gRPC server defaults to localhost binding (good), but warns against exposing it to `0.0.0.0` without authentication.

This “trust the user + interactive flow” approach works for careful developers but leaves gaps for autonomous agent loops, untrusted MCP servers, or accidental destructive commands.

Your Go version should **improve on this** significantly while preserving the original’s fast, terminal-first UX. Below is a **detailed security policy design** focused on **MCP tool support** (and extensible to all tools).

### 1. Core Security Principles (Guided by Original + MCP Spec)

- **Least Privilege**: Every MCP server and tool starts with the strictest policy. Users must explicitly relax it.
- **Explicit Consent**: Dangerous or external actions require clear user approval (improving on the original’s sparse gRPC prompts).
- **Defense in Depth**: Combine policy config, runtime checks, sandboxing, and auditing.
- **Transparency**: Log all MCP connections, tool calls, and approvals (opt-in verbose mode).
- **Fail Safe**: Default to “ask” or “block” for high-risk operations.
- **MCP-Specific**: Follow Anthropic’s official MCP Security Best Practices (progressive disclosure, per-tool scoping, input validation, no implicit trust of remote servers).

### 2. Per-Server MCP Security Policies

Add a new section in your config (`settings.json` or dedicated `mcp.json`):

```json
{
  "mcp": {
    "servers": [
      {
        "name": "github",
        "transport": "stdio",
        "command": ["npx", "@modelcontextprotocol/server-github"],
        "env": { "GITHUB_TOKEN": "ghp_..." },
        "securityPolicy": {
          "approvalMode": "ask",           // "always" | "ask" | "block"
          "dangerousTools": ["create_issue", "push_code", "delete_repo"], // explicit list or "*"
          "autoApproveSafeTools": true,
          "maxExecutionTimeSeconds": 60,
          "allowedPaths": ["./**"],        // for file-related tools from this server
          "networkAccess": "limited",      // "none" | "limited" | "full"
          "requireConfirmationForWrite": true
        },
        "autoConnect": true,
        "trustLevel": "medium"             // "low" | "medium" | "high" (affects defaults)
      }
    ]
  }
}
```

**approvalMode options**:
- **always** — Auto-execute (use only for fully trusted local servers; strongly discouraged for remote).
- **ask** — Show TUI confirmation dialog before every tool call (or only dangerous ones). This is the recommended default and improves on the original.
- **block** — Never allow execution (safe default for unknown servers).

**dangerousTools**: Per-server whitelist/blacklist. If not specified, tools are classified as dangerous based on name/description keywords (e.g., containing “delete”, “write”, “push”, “rm”, “sudo”).

### 3. Global + Per-Tool Safety Layers

**Static classification** (applied to all tools, including MCP-adapted ones):
- Safe by default: `FileRead`, `Grep`, `Glob`, read-only MCP tools.
- Dangerous: Anything involving write, delete, shell execution, network outbound (except approved web search), or external auth.

**Runtime Guards** (in `internal/sandbox/` and `internal/mcp/security/`):
- **Path validation**: All file operations (even from MCP servers) restricted to the current project directory + explicit allowedPaths. Reject absolute paths outside cwd unless user-approved.
- **Command sanitization** (for Bash or MCP tools that execute shell): 
  - Simple blacklist for destructive patterns (`rm -rf /`, `sudo`, `> /dev/sda`, etc.).
  - Timeout enforcement (default 30–60s, configurable per server).
  - Run with restricted environment variables (strip sensitive env except explicitly allowed).
- **Network controls**: MCP servers using HTTP/SSE get rate limiting and optional proxy. Block unexpected outbound connections unless `networkAccess: "full"`.
- **Input validation**: Validate JSON Schema arguments from MCP `tools/call` before execution. Reject oversized payloads.

**User Confirmation Flow** (Bubble Tea TUI):
- For any dangerous MCP tool call: Display clear summary → “Tool: github:create_pr\nArgs: ...\nImpact: Creates a pull request on GitHub\n\nApprove? (y/N)”
- Support “approve once”, “approve for this session”, or “always for this tool/server”.
- In headless/gRPC mode: Emit `action_required` event (matching original behavior) and wait for client response.

### 4. MCP-Specific Security Enhancements

- **Server Trust & Discovery**: On first connect, show full server capabilities (tools list) and ask for confirmation. Never auto-approve tools from untrusted sources.
- **Scoped Permissions**: When adapting MCP tools to your `tools.Tool` interface, tag them with the originating server name (e.g., `github:search_repo`). Policy checks always include the server context.
- **Isolation**: Each MCP server runs in its own goroutine with cancellable context. A crashing or malicious server affects only itself.
- **No Implicit Forwarding**: Do not automatically forward auth tokens or credentials to MCP servers unless explicitly configured and confirmed.
- **Logging & Audit**: Optional `--audit-log` flag writes timestamped entries for every MCP connection and tool execution (server name, tool, args summary, approval status, result size).

### 5. Recommended Defaults & User Guidelines

**Safe Defaults** (better than original):
- New MCP servers → `approvalMode: "ask"`, `trustLevel: "low"`.
- Dangerous tools → always require confirmation unless user changes policy.
- Project directory restriction enforced by default.
- Plaintext secrets warning copied from original (plus recommendation to use environment variables or secret managers where possible).

**Usage Guidelines** (add to your README and `/help`):
1. Only connect MCP servers from trusted sources (official Anthropic samples, well-audited community repos).
2. Review the full list of tools a server exposes before approving connection.
3. Use per-server policies to lock down risky servers (e.g., block write tools on a database MCP server).
4. Run the agent in a dedicated project folder or Git worktree when possible.
5. For maximum safety: Combine with OS-level sandboxing (e.g., `firejail` on Linux, macOS App Sandbox, or run inside a container/VM).
6. Never run with `approvalMode: "always"` on untrusted or remote MCP servers.

### 6. Implementation Notes for Your Go Code

- Extend the MCP Manager (`internal/mcp/manager.go`) with a `SecurityPolicy` struct and evaluator.
- In `tool_adapter.go`: When wrapping an MCP tool, attach its policy context so `Execute()` checks approval before calling the server.
- Reuse/extend your existing sandbox package for MCP tool calls.
- Add CLI commands: `/mcp policy <server> [view|set]` and `/mcp audit`.

This design keeps the flexible, powerful feel of the original OpenClaude while closing its main security gaps — especially for the dynamic nature of MCP servers.

Would you like me to:
- Provide the full Go structs for `SecurityPolicy` + evaluator logic?
- Draft the updated README security section?
- Or expand on sandbox implementation details for MCP tools?

Let me know how to proceed!