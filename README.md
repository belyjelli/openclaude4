# OpenClaude v4

Greenfield rewrite of [OpenClaude](https://github.com/Gitlawb/openclaude) (v3 lives in **openclaude3**). This repository holds design docs and a **Go CLI** with multi-provider config, **Ollama**, and Phase 1 tools.

## Build (Go)

Requires Go 1.22+ (see `go.mod`).

```bash
go build -ldflags "-X main.version=0.1.0 -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o openclaude ./cmd/openclaude
```

### Run (OpenAI-compatible)

```bash
export OPENAI_API_KEY=...
./openclaude
```

### Run (local Ollama)

```bash
export OPENCLAUDE_PROVIDER=ollama
export OLLAMA_MODEL=llama3.2   # optional
./openclaude
```

**CLI:** `./openclaude version`, `./openclaude doctor`, `./openclaude --help`  
**Flags:** `--config`, `--provider`, `--model`, `--base-url`  
**In-session:** `/help`, `/provider`, `/clear`, `/exit`

**Config file:** optional `openclaude.yaml` in the current directory or `~/.config/openclaude/` — see [docs/CONFIG.md](./docs/CONFIG.md) and [openclaude.example.yaml](./openclaude.example.yaml).

**Tools:** `FileRead`, `FileWrite`, `FileEdit`, `Bash`, `Grep`, `Glob`, `WebSearch`. Workspace = process working directory (paths cannot escape it).

**Dangerous tools** prompt on stderr unless `OPENCLAUDE_AUTO_APPROVE_TOOLS=1` or `true`.

## Documents

| Doc | Purpose |
|-----|---------|
| [docs/CONFIG.md](./docs/CONFIG.md) | Env, YAML, flags, v3 migration hints |
| [docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md](./docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md) | What v3 does today (baseline) |
| [docs/DESIGN.md](./docs/DESIGN.md) | Target architecture and principles for v4 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased delivery plan |
| [TODO.md](./TODO.md) | Actionable checklist |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Build, test, lint, PR notes |
| [steps/step1.md](./steps/step1.md) | Go bootstrap notes + status |
| [steps/step2.md](./steps/step2.md) | Phase 1 tools + agent loop notes + status |

## Status

**Phase 2 (started)** — Config file discovery, env merge, `openai` + `ollama` providers, `openclaude doctor`, [docs/CONFIG.md](./docs/CONFIG.md). **Not yet:** Gemini/Codex adapters, automatic v3 profile import. Kernel tests use `httptest` against the OpenAI streaming shape (`internal/core/agent_test.go`).

**Phase 1** — Streaming tool loop, registry, sandboxed shell — see [steps/step2.md](./steps/step2.md).

## License

TBD (align with v3 / project policy when code lands).
