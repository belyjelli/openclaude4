# OpenClaude v4 — providers

All built-in backends speak the **OpenAI Chat Completions** HTTP API (`POST /v1/chat/completions`, optional tools, SSE streaming). The Go client is `github.com/sashabaranov/go-openai` with different `BaseURL` / `model` / auth per provider.

| Provider | When to use | Auth | Base URL (default) | Model source |
|----------|-------------|------|--------------------|--------------|
| **openai** (default) | OpenAI or any OpenAI-compatible host (DeepSeek, Groq, Azure OpenAI with compat layer, etc.) | `OPENAI_API_KEY` | `https://api.openai.com/v1` or `OPENAI_BASE_URL` / `provider.base_url` | `OPENAI_MODEL` / `provider.model` (default `gpt-4o-mini`) |
| **ollama** | Local models via Ollama | None (placeholder key in client) | `{OLLAMA_HOST}/v1` (host default `http://127.0.0.1:11434`) | `OLLAMA_MODEL` / `ollama.model` / `provider.model` (default `llama3.2`) |
| **gemini** | Google Gemini via **OpenAI-compatible** endpoint | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | `https://generativelanguage.googleapis.com/v1beta/openai` (override with `GEMINI_BASE_URL` / `gemini.base_url`) | `GEMINI_MODEL` / `gemini.model` / `provider.model` (default `gemini-2.0-flash`) |
| **codex** | Reserved | — | — | Not implemented yet (`openclaude` will error at startup). |

## Wiring in code

- Factory: [`internal/providers/runtime.go`](../internal/providers/runtime.go) (`NewStreamClient`).
- Implementations: [`internal/providers/openaicomp/client.go`](../internal/providers/openaicomp/client.go) (`New`, `NewOllama`, `NewGemini`).
- Config getters: [`internal/config/config.go`](../internal/config/config.go).

## Tests

The agent loop is covered with **httptest** SSE mocks for multiple **model id** strings (OpenAI, Ollama tag, Gemini id) in [`internal/core/agent_test.go`](../internal/core/agent_test.go) (`TestRunUserTurn_OpenAICompatiblePerProviderModel`). The wire format is the same; only the configured model name and API key differ.

## See also

- [CONFIG.md](./CONFIG.md) — precedence, env vars, YAML, v3 profile merge
- [adr/0001-go-tooling-and-config.md](./adr/0001-go-tooling-and-config.md) — compatibility decisions
