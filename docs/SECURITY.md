# Security notes (OpenClaude v4)

This document summarizes how the Go CLI limits filesystem and shell access today. It is **not** a full threat model.

## Workspace boundary

File tools (`FileRead`, `FileWrite`, `FileEdit`) and directory-scoped tools (`Grep` search path, `Glob` root) resolve paths against a **workspace root**: the process working directory at startup, unless overridden via context (tests use an explicit root).

- Relative paths are interpreted under that root.
- Absolute paths are allowed only if they still lie **inside** the same root after `filepath.Clean`; otherwise the call fails with an error that mentions escaping the workspace.
- Logic lives in `internal/tools/paths.go` (`resolveUnderWorkdir`). Regression tests are in `internal/tools/workspace_boundary_test.go`.

**Symlinks:** resolution does **not** use `filepath.EvalSymlinks`. A symlink *inside* the workspace can still point **outside** it; the kernel does not walk that link when checking the prefix, so reads/writes may follow the link to an unintended location. Treat untrusted symlinks in the workspace as out of scope for the simple prefix check, or extend the resolver with evaluated paths if you need stronger guarantees.

## Dangerous tools

`Bash`, `FileWrite`, and `FileEdit` are marked dangerous. The stdin REPL prompts for confirmation unless `OPENCLAUDE_AUTO_APPROVE_TOOLS` is set to `1` or `true` (development convenience only).

Shell commands run with a **timeout** and workspace-oriented working directory; they are still **full shell** invocations—users should not auto-approve in untrusted environments.

**Bash timeouts (`internal/tools/bash.go`):** default **120 seconds** per invocation; the model may pass `timeout_seconds` (capped at **600**). The timer is a `context.WithTimeout` around `sandbox.RunShell` (see `internal/sandbox/sandbox.go`). There is **no** extra OpenClaude rate limit on how often `Bash` may run; only the per-call timeout and the agent iteration cap below apply.

**MCP tools** (from `mcp.servers` in config) run inside **separate child processes** you configure. They are **not** sandboxed by OpenClaude’s workspace rules unless the server enforces that. With `approval: ask` (default), each MCP tool invocation uses the same confirmation path as other dangerous tools; `always` / `never` skip that prompt—only use those for servers you trust.

The **`Task`** tool starts a **nested agent loop** with the same provider and tools (including MCP); the child registry **drops `Task`** so the sub-agent cannot recurse. It uses the same confirm hook as other dangerous tools, does **not** stream sub-agent text to the terminal (only the final summary returns to the parent turn), and is **dangerous**—avoid `OPENCLAUDE_AUTO_APPROVE_TOOLS` with untrusted configs.

## Network

`WebSearch`, `WebFetch`, and LLM providers perform outbound HTTP(S). Use API keys and base URLs you trust.

### Built-in HTTP tool (`WebSearch`)

Implemented in `internal/tools/web_search.go`:

- **HTTP client timeout:** **15 seconds** per request (`http.Client{Timeout: 15 * time.Second}`) to DuckDuckGo’s JSON API (`https://api.duckduckgo.com/`).
- **Response body read cap:** **1 MiB** (`io.LimitReader` on the response body).
- **Output size:** related-topic lines stop once the built summary reaches about **8000** bytes (loop break on `b.Len() > 8000`).

OpenClaude does **not** add an application-level request rate limit for `WebSearch`; remote services may still throttle by IP or policy.

### Built-in HTTP tool (`WebFetch`)

Implemented in `internal/tools/web_fetch.go`:

- **HTTP client timeout:** **20 seconds** per request (including redirects).
- **Redirects:** at most **5**; each redirect target is validated the same way as the original URL.
- **Allowed URLs:** **http** and **https** only; URLs with embedded **userinfo** (credentials) are rejected.
- **SSRF mitigation:** **localhost** hostnames and **loopback / private / link-local** IP addresses are rejected, including when returned from **DNS** resolution for a hostname (if any resolved address is disallowed, the fetch fails). This reduces casual misuse but is **not** a complete SSRF barrier (DNS rebinding, exotic encodings, and future address semantics may still warrant caution).
- **Response body read cap:** **2 MiB** (`io.LimitReader`); larger bodies return an error.
- **Text extraction:** **HTML** (and XHTML content types, or HTML-like sniff) is parsed with `golang.org/x/net/html` and reduced to plain text (script/style/svg/template subtrees skipped). Non-HTML bodies must be **valid UTF-8** (JSON, plain text, etc.).
- **Output cap:** extracted text is truncated to **`max_chars`** (default **80000**, maximum **200000**) with a trailing notice.

OpenClaude does **not** add an application-level request rate limit for `WebFetch`.

### LLM provider HTTP (`openaicomp` / go-openai)

Chat streaming uses `github.com/sashabaranov/go-openai` with `sdk.DefaultConfig` (`internal/providers/openaicomp/client.go`). The library’s default `HTTPClient` is `&http.Client{}` with **no** `Timeout` field set, so OpenClaude does **not** currently impose a fixed wall-clock HTTP timeout on completions—calls end when the stream finishes, the remote closes, an error occurs, or the process exits (root command `context` from Cobra, typically no deadline). Combine with the agent iteration limit so a single user line cannot loop tools forever (see below).

### `openclaude doctor` reachability

`internal/providers/ping.go` uses **`pingHTTP`**: **`http.Client{Timeout: 3 * time.Second}`** for quick GETs (e.g. Ollama `/api/tags`, Gemini models list). This is **only** for the doctor banner, not for chat completions.

### Agent and `Task` iteration caps (not HTTP, but bounds work per user line)

- **Main agent** (`internal/core/agent.go`): at most **24** model↔tool rounds per user message (`defaultMaxIterations`), unless `Agent.MaxIterations` is set positive.
- **`Task` sub-agent** (`internal/core/task_tool.go`): default **12** rounds; `max_iterations` applies only when it is a positive number **strictly below 1000**, then is **capped at 24** (same constant as the main agent). Values **≥ 1000** leave the default **12** in effect.

### MCP tool calls

MCP tools use `session.CallTool(ctx, …)` with the same chat `context` (`internal/mcpclient/manager.go`). OpenClaude does **not** set a separate timeout or rate limit for MCP beyond that context and whatever the MCP SDK/server does.

## On-disk sessions

The CLI saves chat transcripts as **JSON** under the session directory (default `~/.local/share/openclaude/sessions/`, or `session.dir` / `OPENCLAUDE_SESSION_DIR`). Files include **message text, tool arguments, and tool outputs** from the model loop — treat them as **sensitive** (credentials, file paths, code). Use `--no-session` or `OPENCLAUDE_NO_SESSION=true` to disable persistence. A `last_session_id` file records the last session written for `--resume`. See [CONFIG.md](./CONFIG.md#sessions-and-compaction).

## Transcript and log redaction

Kernel events delivered to [`Agent.OnEvent`](../internal/core/agent.go) (TUI, future session exports) pass through [`RedactEventForLog`](../internal/core/redact.go) so common secret shapes are replaced with `[REDACTED]` before handlers run. **API traffic to the model is unchanged**; only the observable/logged event copy is scrubbed. Dangerous-tool confirmation on the plain REPL prints JSON arguments via the same redaction helper.

Covered patterns include `Bearer` tokens, `Authorization:` header lines, well-known `*_API_KEY=…` env fragments, OpenAI- and Google-style key prefixes, JSON string fields whose names look like `api_key` / `access_token` / `password` / etc., tool-argument map keys with similar names (values fully redacted), and long base64-like runs (80+ characters) in free text. This is **heuristic**: novel encodings, short secrets, or unusual field names may still appear verbatim; long alphanumeric text can occasionally be over-redacted.

## Ongoing work

See [TODO.md](../TODO.md): dependency update policy, rate limits/timeouts documentation, and other quality items.
