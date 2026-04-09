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
  name: openai      # openai | ollama | gemini | github | openrouter | codex (codex reserved — client build fails; see Validation)
  model: gpt-4o-mini
  base_url: ""      # optional; OpenAI-compatible endpoints only

openrouter:
  api_key: ""       # prefer OPENROUTER_KEY or OPENROUTER_API_KEY
  model: openai/gpt-4o-mini
  provider: ""      # optional OPENROUTER_PROVIDER — filters /model catalog to "<slug>/…"

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
      # Prefer bunx -y for npm MCP servers (fast install/run; same packages as npx).
      command: ["bunx", "-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/root"]
      approval: ask        # ask | always | never — ask = confirm like other dangerous tools
      # env:                 # optional extra KEY: value pairs for the child process
      #   FOO: bar
```

### MCP servers

Each entry runs **`command`** as a subprocess; OpenClaude talks to it over **stdin/stdout** (MCP JSON-RPC). **`name`** must be unique; it appears in tool names as `mcp_<name>__<tool>`.

- **`approval`**: `ask` (default) treats every tool from that server as dangerous (stdin confirmation unless `OPENCLAUDE_AUTO_APPROVE_TOOLS`). `always` and `never` skip that prompt (use only for servers you trust).

Failed servers are skipped with a message on stderr; chat still starts if built-in tools are enough.

**CLI:** `openclaude mcp add --name <id> --exec <argv1> --exec <argv2> ...` appends a server to the config file [`config.WritableConfigPath`](../internal/config/mcp_configfile.go) would choose (same rules as loading: explicit `--config`, else first `openclaude.{yaml,yml,json}` on the search path, else `~/.config/openclaude/openclaude.yaml`). For npm-published MCP servers, **`--bunx`** prepends `bunx -y` (recommended); you still pass `--exec` for the package name and arguments. Local scripts: `--exec bun --exec run --exec ./server.ts`. Use `--approval always|never|ask`, `--dry-run` to preview. Rewriting YAML drops comments.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `OPENCLAUDE_PROVIDER` | `openai` (default), `ollama`, `gemini`, `github`, `openrouter`, or `codex` |
| `OPENAI_API_KEY` | API key for OpenAI-compatible APIs |
| `OPENAI_BASE_URL` | Custom base URL (OpenAI-compatible); if it targets OpenRouter (`openrouter.ai`), `OPENROUTER_KEY` may be used when this key is empty |
| `OPENAI_MODEL` | Model id (bound to `provider.model`) |
| `OLLAMA_HOST` | Ollama server URL (default `http://127.0.0.1:11434`) |
| `OLLAMA_MODEL` | Ollama model tag |
| `GEMINI_API_KEY` / `GOOGLE_API_KEY` | Gemini API key |
| `GEMINI_MODEL` | Gemini model id |
| `GEMINI_BASE_URL` | Override Gemini OpenAI-compatible base URL |
| `OPENROUTER_KEY` / `OPENROUTER_API_KEY` | OpenRouter API key (provider `openrouter`, or `openai` with OpenRouter base URL) |
| `OPENROUTER_MODEL` | Model id for provider `openrouter` (bound to `openrouter.model`) |
| `OPENROUTER_PROVIDER` | Optional filter for `/model` catalog (`openrouter.provider`) |
| `OPENCLAUDE_AUTO_APPROVE_TOOLS` | `1` / `true` to skip dangerous-tool prompts (dev only) |

**Richer web scrape (no Firecrawl):** install [spider-rs `spider_cli`](https://github.com/spider-rs/spider) so `spider` is on `PATH`; the optional **`SpiderScrape`** tool is then registered (see [SECURITY.md](./SECURITY.md#optional-subprocess-tool-spiderscrape)).

### Sessions and compaction

| Variable / flag | Purpose |
|-----------------|--------|
| `--session` / `OPENCLAUDE_SESSION` | Fixed session id (file name base under the session directory) |
| `--resume` / `OPENCLAUDE_RESUME` | Open the last saved session (`last_session_id` or newest `*.json`) |
| `--list-sessions` | Print saved sessions and exit |
| `--print` / `-p` | **One-shot (non-interactive):** run a single user message and exit. The **final assistant text** is printed to stdout (streaming is discarded). Prompt is the flag value; use **`-p -`** to read the prompt from stdin. Incompatible with `--tui` / `OPENCLAUDE_TUI`. **Dangerous tools:** unless `OPENCLAUDE_AUTO_APPROVE_TOOLS` is set, each dangerous tool is **skipped** (stderr explains how to enable). Sessions apply like the REPL (`--session`, `--resume`, `--no-session`). |
| `--no-session` / `OPENCLAUDE_NO_SESSION` | Do not read or write session files |
| `OPENCLAUDE_SESSION_DIR` / `session.dir` | Override session directory (default `~/.local/share/openclaude/sessions`) |
| `OPENCLAUDE_SESSION_COMPACT_TOKEN_THRESHOLD` / `session.compact_token_threshold` | Rough token estimate above which the next user turn auto-compacts (0 = off) |
| `OPENCLAUDE_SESSION_SUMMARIZE_OVER_THRESHOLD` / `session.summarize_over_threshold` | If threshold tripped, call the model for a summary instead of lossy tail trim (falls back to trim on failure) |
| `OPENCLAUDE_SESSION_COMPACT_KEEP_MESSAGES` / `session.compact_keep_messages` | Tail size for `/compact` and lossy auto-compact (default 24 after system) |

**Startup banner:**

| Variable | Meaning |
| -------- | ------- |
| `OPENCLAUDE_NO_SPLASH` | Set to `1`, `true`, or `yes` to skip the gradient ANSI splash and use the short plain header instead (same style as non-TTY / `CI`). Applies to the plain REPL (stderr) and to the **TUI** transcript (scrollable area). |

**TUI (`--tui` / `OPENCLAUDE_TUI=1`):**

With `--tui`, the startup banner (ANSI or plain, per the rules above) is shown **inside the transcript** at the top, then the TUI hint line — not as a separate stderr block before the full-screen UI.

| Variable | Purpose |
|----------|--------|
| `OPENCLAUDE_TUI_TOOL_PREVIEW` | Max UTF-8 **runes** of each tool’s stdout shown in the transcript (default 4000; `0` = default; invalid = 400) |
| `OPENCLAUDE_TUI_MARKDOWN` | Set `0` / `false` / `no` for plain assistant text while **streaming** and on **finished** turns (default: Goldmark + Chroma terminal markdown). While streaming, an **unclosed** fenced code block (` ``` ` / `~~~`) is shown **plain** from its opening line until the closing fence, so the prefix above it can still render as markdown. |

**Running registry:** each interactive chat writes `<session-dir>/running/<pid>.json` (removed on clean exit). Inspect with **`openclaude sessions`** or **`/session running`** in the REPL/TUI.

**Skills (SKILL.md):**

| Key / env | Purpose |
|-----------|--------|
| `skills.dirs` (YAML list) | Extra directories to scan for subfolders containing `SKILL.md` |
| `OPENCLAUDE_SKILLS_DIRS` | Comma-separated skill roots (merged with config) |
| *(defaults)* | `./.openclaude/skills` and `~/.local/share/openclaude/skills` if those directories exist |

**Task sub-agent model override:**

| Key / env | Purpose |
|-----------|--------|
| `agent_routing.task_model` | Model id for **Task** tool sub-runs (same provider/endpoint as the main client) |
| `OPENCLAUDE_AGENT_TASK_MODEL` | Env alias for `agent_routing.task_model` |

**Vision (OpenAI-style `image_url` parts):**

| Flag | Purpose |
|------|--------|
| `--image-url` | Repeatable HTTP(S) URL attached to the **first** non-slash user message (REPL/TUI) or combined with `-p` |
| `--image-file` | Repeatable local file path (read as base64 `data:` URL; max 8 MiB per file) |

Ollama exposes an OpenAI-compatible chat API at `{OLLAMA_HOST}/v1` ([Ollama OpenAI docs](https://github.com/ollama/ollama/blob/main/docs/openai.md)).

## v3 migration notes

OpenClaude v3 uses **`~/.openclaude-profile.json`** (and optionally a copy in the project directory) plus **`settings.json`**.

| v3 artifact | v4 behavior |
|-------------|-------------|
| `.openclaude-profile.json` | **Merged** (low precedence). Supported `profile` values include `openai`, `ollama`, `gemini`, `codex`, and `atomic-chat` (mapped to OpenAI). Env entries in the JSON are mapped into the same keys as YAML/env. |
| `settings.json` | **Not** read. Relevant options should be translated into `openclaude.yaml` or env vars. |

For manual mapping: API keys → `OPENAI_*` / `GEMINI_*` / YAML; custom OpenAI base → `OPENAI_BASE_URL` or `provider.base_url`; local Ollama → `OPENCLAUDE_PROVIDER=ollama` and `OLLAMA_MODEL`.

## Validation

Unknown `provider.name` values are rejected when the chat (or serve) command loads config (`config.Validate()`). The reserved name **`codex`** fails validation with the same `ErrCodexNotImplemented` sentinel used by [`NewStreamClient`](../internal/providers/runtime.go) ([`internal/providererrs`](../internal/providererrs/codex.go)). Run `openclaude doctor` to see validation and client construction in one place.

## In-session slash commands (REPL)

Handled in the chat loop (not config keys): `/help`, `/onboard` / `/setup`, `/provider` (and **`/provider wizard`** for interactive YAML/env setup in the plain REPL; in `--tui`, wizard prints a static copy-paste guide), `/mcp list`, `/mcp doctor`, `/mcp help`, `/skills list`, `/skills read <name>`, `/session` (show, list, **running** / **ps**, load, new, save), `/compact` (lossy transcript trim: keeps system + last N messages, default 24; override via `session.compact_keep_messages`), `/clear`, `/exit`, `/quit`. From the shell: **`openclaude sessions`** (saved sessions + running registry), **`openclaude mcp list`** (config only), **`openclaude mcp doctor`** (connect + list tools; exit 1 if any server fails), **`openclaude mcp add`** (append a `mcp.servers` entry — repeats `--exec` per argv token; **`--dry-run`** to preview; YAML comments are not preserved on rewrite).

## Diagnostics

```bash
openclaude doctor
```

Reports Go version, `rg` and `spider` on `PATH`, active provider/model, reachability hints, and whether a chat client can be constructed.

## gRPC (`openclaude serve`)

Headless **`openclaude.v4.AgentService`** (see [`internal/grpc/README.md`](../internal/grpc/README.md)):

| Mechanism | Purpose |
|-----------|---------|
| **`openclaude serve`** | Start the gRPC server after the same config validation as chat. |
| **`OPENCLAUDE_GRPC_ADDR`** | Listen address if `--listen` is not set (default `:50051`). |
| **`--listen`** | Overrides `OPENCLAUDE_GRPC_ADDR`. |

Session persistence for gRPC uses the same **`session.*` / `OPENCLAUDE_NO_SESSION`** rules as the REPL. Clients set **`ChatRequest.session_id`** to load/save a named session; see the proto and gRPC README.

**Vision over gRPC:** `ChatRequest.image_url` (repeated) and `image_inline` (raw bytes + `mime_type`) are optional; you may send images with empty `user_text` (server uses the same placeholder text as the CLI). Limits: 16 images per turn, 8 MiB per inline attachment — see [`internal/grpc/proto/openclaude.proto`](../internal/grpc/proto/openclaude.proto).

## Timeouts, iteration limits, and HTTP behavior

There are **no** YAML/env knobs for these today; values are fixed in Go code. Rationale and security notes live in [SECURITY.md](./SECURITY.md#network).

| Area | Behavior | Code |
|------|----------|------|
| **`Bash`** | Per invocation: default **120 s** wall clock; optional `timeout_seconds` on the tool args, max **600 s**. | `internal/tools/bash.go`, `internal/sandbox/sandbox.go` |
| **`WebSearch`** | DuckDuckGo instant-answer JSON API (**15 s**, **1 MiB** cap; related topics stop around **~8000** bytes). | `internal/tools/web_search.go` |
| **`WebFetch`** | HTTP(S) GET **20 s**, **5** redirects, **2 MiB** body; SSRF-minded URL rules; HTML→text; `max_chars` truncation. See [SECURITY.md](./SECURITY.md#built-in-http-tool-webfetch). | `internal/tools/web_fetch.go` |
| **`SpiderScrape`** | Optional: registered only if `spider` is on `PATH`. Runs `spider … scrape` (**60 s** timeout); same URL rules as WebFetch; output cap per `max_chars`. See [SECURITY.md](./SECURITY.md#optional-subprocess-tool-spiderscrape). | `internal/tools/spider_scrape.go` |
| **Main chat agent** | Up to **24** model↔tool rounds per user line (unless code sets `Agent.MaxIterations` > 0). | `internal/core/agent.go` (`defaultMaxIterations`) |
| **`Task` tool** | Sub-agent default **12** rounds; `max_iterations` only honored if **0 < value < 1000**, then capped at **24**. | `internal/core/task_tool.go` |
| **LLM HTTP** | Uses `go-openai` `DefaultConfig` with default `http.Client{}` (**no** `Timeout` in our wiring). Streams end on completion, error, or process exit—not on a fixed OpenClaude HTTP deadline. | `internal/providers/openaicomp/client.go` |
| **`openclaude doctor` ping** | Reachability GETs use **3 s** HTTP client timeout. | `internal/providers/ping.go` |
| **MCP `CallTool`** | Uses the chat command `context`; no separate OpenClaude timeout or rate limit. | `internal/mcpclient/manager.go` |

## Further reading

- [PROVIDERS.md](./PROVIDERS.md) — OpenAI / Ollama / Gemini matrix and code pointers
- [ADR 0001: Go tooling and config compatibility](./adr/0001-go-tooling-and-config.md)
- [SECURITY.md](./SECURITY.md) — workspace, dangerous tools, and [network/timeouts](./SECURITY.md#network)
