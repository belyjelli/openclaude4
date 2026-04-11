# OpenClaude v4 — providers

All built-in backends speak the **OpenAI Chat Completions** HTTP API (`POST /v1/chat/completions`, optional tools, SSE streaming). The Go client is `github.com/sashabaranov/go-openai` with different `BaseURL` / `model` / auth per provider.

| Provider | When to use | Auth | Base URL (default) | Model source |
|----------|-------------|------|--------------------|--------------|
| **openai** (default) | OpenAI or any OpenAI-compatible host (DeepSeek, Groq, OpenRouter, Azure OpenAI with compat layer, etc.) | `OPENAI_API_KEY` (or `OPENROUTER_KEY` when `OPENAI_BASE_URL` / `provider.base_url` targets `openrouter.ai`) | `https://api.openai.com/v1` or `OPENAI_BASE_URL` / `provider.base_url` | `--model` / yaml `provider.model`, then `OPENAI_MODEL` (default `gpt-4o-mini`) |
| **ollama** | Local models via Ollama | None (placeholder key in client) | `{OLLAMA_HOST}/v1` (host default `http://127.0.0.1:11434`) | `OLLAMA_MODEL` / `ollama.model`, then `--model` / yaml `provider.model` (default `llama3.2`). `OPENAI_MODEL` is ignored. |
| **gemini** | Google Gemini via **OpenAI-compatible** endpoint | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | `https://generativelanguage.googleapis.com/v1beta/openai` (override with `GEMINI_BASE_URL` / `gemini.base_url`) | `GEMINI_MODEL` / `gemini.model`, then `--model` / yaml `provider.model` (default `gemini-2.0-flash`). `OPENAI_MODEL` is ignored. |
| **github** | GitHub Models (Azure-hosted OpenAI-compatible API) | `GITHUB_TOKEN` or `GITHUB_PAT` | `{GITHUB_BASE_URL}` (omit for default; pattern: `https://<region>.models.ai.azure.com`) | `GITHUB_MODEL` / `github.model`, then `--model` / yaml `provider.model` (default `gpt-4o`). `OPENAI_MODEL` is ignored. |
| **openrouter** | [OpenRouter](https://openrouter.ai/) as a named provider | `OPENROUTER_KEY` or `OPENROUTER_API_KEY` | `https://openrouter.ai/api/v1` (override with `OPENAI_BASE_URL` / `provider.base_url`) | `OPENROUTER_MODEL` / `openrouter.model`, then `--model` / yaml `provider.model` (default `openai/gpt-4o-mini`). `OPENAI_MODEL` is ignored. |
| **codex** | Reserved | — | — | Not implemented: `config.Validate()` and building the stream client return `ErrCodexNotImplemented` (chat, `serve`, `doctor`). |

**Model catalog (`/model`):** With `OPENROUTER_KEY` set, OpenClaude uses the official [OpenRouter Go SDK](https://github.com/OpenRouterTeam/go-sdk) to list models (optional filter `OPENROUTER_PROVIDER`). With provider **openrouter**, the catalog uses the same key.

## MCP (Model Context Protocol)

Optional **stdio** MCP servers are listed under `mcp.servers` in `openclaude.yaml` (see [CONFIG.md](./CONFIG.md)). At chat startup, OpenClaude connects each server, lists tools, and registers them on the same registry as built-ins. Exposed OpenAI function names are `mcp_<server>__<tool>` (see [`OpenAIToolName`](../internal/mcpclient/schema.go)).

Use `/mcp list` in the REPL and `openclaude doctor` to inspect configuration. Failed servers are skipped with a stderr line; chat still runs with built-in tools only.

## Wiring in code

- Factory: [`internal/providers/runtime.go`](../internal/providers/runtime.go) (`NewStreamClient`).
- Implementations: [`internal/providers/openaicomp/client.go`](../internal/providers/openaicomp/client.go) (`New`, `NewOllama`, `NewGemini`, `NewOpenRouter`), [`internal/providers/openaicomp/github.go`](../internal/providers/openaicomp/github.go) (`NewGitHubModels`). OpenRouter catalog: [`internal/providers/openrouter_model_list.go`](../internal/providers/openrouter_model_list.go).
- Config getters: [`internal/config/config.go`](../internal/config/config.go).

## Tests

The agent loop is covered with **httptest** SSE mocks for multiple **model id** strings (OpenAI, Ollama tag, Gemini id) in [`internal/core/agent_test.go`](../internal/core/agent_test.go) (`TestRunUserTurn_OpenAICompatiblePerProviderModel`). The wire format is the same; only the configured model name and API key differ.

Structured **kernel events** (streaming deltas, tool calls, errors) are emitted via [`Agent.OnEvent`](../internal/core/agent.go); see [`internal/core/event.go`](../internal/core/event.go) and [`event_harness_test.go`](../internal/core/event_harness_test.go).

## See also

- [CONFIG.md](./CONFIG.md) — precedence, env vars, YAML, v3 profile merge
- [adr/0001-go-tooling-and-config.md](./adr/0001-go-tooling-and-config.md) — compatibility decisions
