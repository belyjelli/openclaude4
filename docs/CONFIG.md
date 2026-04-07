# OpenClaude v4 — configuration

## Precedence (what wins when keys overlap)

Sources are merged in two phases.

### Phase A — file merge order (first merged = weakest)

1. **v3 profile** — `.openclaude-profile.json` in the current working directory, then `$HOME/.openclaude-profile.json` (first file found). See `internal/config/profile_v3.go`.
2. **v4 config file** — `openclaude.{yaml,yml,json}` from `./` then `~/.config/openclaude/`, **or** the path passed as `--config`. The first existing file wins among the search paths.

Later merges override earlier ones for the same keys (so the v4 file overrides the v3 profile).

### Phase B — viper lookup order (on each read)

When the code calls `viper.Get*` for a key, **spf13/viper** applies (highest wins):

1. Explicit `viper.Set` (internal use only)
2. **CLI flags** (`--provider`, `--model`, `--base-url`)
3. **Environment variables** (see below)
4. Merged config from phase A
5. Defaults implied by helpers in `internal/config/config.go` (e.g. default model names)

**Summary:** `flags → env → openclaude.yaml (or --config) → .openclaude-profile.json → defaults`.

## Config file search

Unless you pass `--config /path/to/file`:

1. `./openclaude.yaml` (also `.yml`, `.json`)
2. `~/.config/openclaude/openclaude.{yaml,yml,json}`

The first existing file wins.

## Example (`openclaude.yaml`)

```yaml
openai:
  api_key: sk-...   # prefer OPENAI_API_KEY in env instead of committing secrets

provider:
  name: openai      # openai | ollama | gemini | codex (codex not implemented yet)
  model: gpt-4o-mini
  base_url: ""      # optional; OpenAI-compatible endpoints only

ollama:
  host: http://127.0.0.1:11434
  model: llama3.2

gemini:
  api_key: ""       # prefer GEMINI_API_KEY or GOOGLE_API_KEY
  model: gemini-2.0-flash
  base_url: ""      # optional; default is Google's OpenAI-compat Gemini endpoint

# Optional: Model Context Protocol (stdio subprocesses). See docs/SECURITY.md.
mcp:
  servers:
    - name: fs
      command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/root"]
      approval: ask        # ask | always | never — ask = confirm like other dangerous tools
      # env:                 # optional extra KEY: value pairs for the child process
      #   FOO: bar
```

### MCP servers

Each entry runs **`command`** as a subprocess; OpenClaude talks to it over **stdin/stdout** (MCP JSON-RPC). **`name`** must be unique; it appears in tool names as `mcp_<name>__<tool>`.

- **`approval`**: `ask` (default) treats every tool from that server as dangerous (stdin confirmation unless `OPENCLAUDE_AUTO_APPROVE_TOOLS`). `always` and `never` skip that prompt (use only for servers you trust).

Failed servers are skipped with a message on stderr; chat still starts if built-in tools are enough.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `OPENCLAUDE_PROVIDER` | `openai` (default), `ollama`, `gemini`, or `codex` |
| `OPENAI_API_KEY` | API key for OpenAI-compatible APIs |
| `OPENAI_BASE_URL` | Custom base URL (OpenAI-compatible) |
| `OPENAI_MODEL` | Model id (bound to `provider.model`) |
| `OLLAMA_HOST` | Ollama server URL (default `http://127.0.0.1:11434`) |
| `OLLAMA_MODEL` | Ollama model tag |
| `GEMINI_API_KEY` / `GOOGLE_API_KEY` | Gemini API key |
| `GEMINI_MODEL` | Gemini model id |
| `GEMINI_BASE_URL` | Override Gemini OpenAI-compatible base URL |
| `OPENCLAUDE_AUTO_APPROVE_TOOLS` | `1` / `true` to skip dangerous-tool prompts (dev only) |

Ollama exposes an OpenAI-compatible chat API at `{OLLAMA_HOST}/v1` ([Ollama OpenAI docs](https://github.com/ollama/ollama/blob/main/docs/openai.md)).

## v3 migration notes

OpenClaude v3 uses **`~/.openclaude-profile.json`** (and optionally a copy in the project directory) plus **`settings.json`**.

| v3 artifact | v4 behavior |
|-------------|-------------|
| `.openclaude-profile.json` | **Merged** (low precedence). Supported `profile` values include `openai`, `ollama`, `gemini`, `codex`, and `atomic-chat` (mapped to OpenAI). Env entries in the JSON are mapped into the same keys as YAML/env. |
| `settings.json` | **Not** read. Relevant options should be translated into `openclaude.yaml` or env vars. |

For manual mapping: API keys → `OPENAI_*` / `GEMINI_*` / YAML; custom OpenAI base → `OPENAI_BASE_URL` or `provider.base_url`; local Ollama → `OPENCLAUDE_PROVIDER=ollama` and `OLLAMA_MODEL`.

## Validation

Invalid `provider.name` values are rejected at chat startup (`config.Validate()`). Run `openclaude doctor` to see validation and client errors in one place.

## In-session slash commands (REPL)

Handled in the chat loop (not config keys): `/help`, `/provider`, `/mcp list`, `/compact` (lossy transcript trim: keeps system + last 24 messages), `/clear`, `/exit`, `/quit`.

## Diagnostics

```bash
openclaude doctor
```

Reports Go version, `rg` on `PATH`, active provider/model, reachability hints, and whether a chat client can be constructed.

## Timeouts, iteration limits, and HTTP behavior

There are **no** YAML/env knobs for these today; values are fixed in Go code. Rationale and security notes live in [SECURITY.md](./SECURITY.md#network).

| Area | Behavior | Code |
|------|----------|------|
| **`Bash`** | Per invocation: default **120 s** wall clock; optional `timeout_seconds` on the tool args, max **600 s**. | `internal/tools/bash.go`, `internal/sandbox/sandbox.go` |
| **`WebSearch`** | HTTP client timeout **15 s**; response body read cap **1 MiB**; output truncated once the text builder exceeds **~8000** bytes for related topics. No OpenClaude-side request rate limit. | `internal/tools/web_search.go` |
| **Main chat agent** | Up to **24** model↔tool rounds per user line (unless code sets `Agent.MaxIterations` > 0). | `internal/core/agent.go` (`defaultMaxIterations`) |
| **`Task` tool** | Sub-agent default **12** rounds; `max_iterations` only honored if **0 < value < 1000**, then capped at **24**. | `internal/core/task_tool.go` |
| **LLM HTTP** | Uses `go-openai` `DefaultConfig` with default `http.Client{}` (**no** `Timeout` in our wiring). Streams end on completion, error, or process exit—not on a fixed OpenClaude HTTP deadline. | `internal/providers/openaicomp/client.go` |
| **`openclaude doctor` ping** | Reachability GETs use **3 s** HTTP client timeout. | `internal/providers/ping.go` |
| **MCP `CallTool`** | Uses the chat command `context`; no separate OpenClaude timeout or rate limit. | `internal/mcpclient/manager.go` |

## Further reading

- [PROVIDERS.md](./PROVIDERS.md) — OpenAI / Ollama / Gemini matrix and code pointers
- [ADR 0001: Go tooling and config compatibility](./adr/0001-go-tooling-and-config.md)
- [SECURITY.md](./SECURITY.md) — workspace, dangerous tools, and [network/timeouts](./SECURITY.md#network)
