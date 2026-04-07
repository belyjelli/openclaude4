# OpenClaude v4 — configuration (Phase 2)

Configuration is merged in this order (later wins):

1. Defaults baked into the binary  
2. Config file (if found)  
3. Environment variables  
4. CLI flags (`--provider`, `--model`, `--base-url`, …)

## Config file search

Unless you pass `--config /path/to/file`:

1. `./openclaude.yaml` (also `.yml`, `.json`)
2. `~/.config/openclaude/openclaude.{yaml,yml,json}`

The first existing file wins.

## Example (`openclaude.yaml`)

```yaml
openai:
  api_key: sk-...   # prefer env OPENAI_API_KEY instead of committing secrets

provider:
  name: openai      # or ollama
  model: gpt-4o-mini
  base_url: ""      # optional; OpenAI-compatible only

ollama:
  host: http://127.0.0.1:11434
  model: llama3.2
```

## Environment variables

| Variable | Purpose |
|----------|---------|
| `OPENCLAUDE_PROVIDER` | `openai` (default) or `ollama` |
| `OPENAI_API_KEY` | API key for OpenAI-compatible APIs |
| `OPENAI_BASE_URL` | Custom base URL (OpenAI-compatible) |
| `OPENAI_MODEL` | Model id (bound to `provider.model`) |
| `OLLAMA_HOST` | Ollama server URL (default `http://127.0.0.1:11434`) |
| `OLLAMA_MODEL` | Ollama model tag |
| `OPENCLAUDE_AUTO_APPROVE_TOOLS` | `1` / `true` to skip dangerous-tool prompts |

Ollama exposes an OpenAI-compatible chat API at `{OLLAMA_HOST}/v1` ([Ollama docs](https://github.com/ollama/ollama/blob/main/docs/openai.md)).

## v3 migration notes

OpenClaude v3 (TypeScript) uses `~/.openclaude-profile.json` and `settings.json`. v4 Go does **not** read those files yet; map settings manually:

- Remote API key → `OPENAI_API_KEY` or `openai.api_key` in YAML  
- Custom endpoint → `OPENAI_BASE_URL` or `provider.base_url`  
- Local Ollama → `OPENCLAUDE_PROVIDER=ollama` and `OLLAMA_MODEL`

## Diagnostics

```bash
openclaude doctor
```

Reports Go version, `rg` on `PATH`, active provider/model, and a quick reachability check for Ollama when selected.
