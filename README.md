# OpenClaude v4

Greenfield rewrite of [OpenClaude](https://github.com/Gitlawb/openclaude) (v3 lives in **openclaude3**). This repository holds design docs and a **Go CLI** with multi-provider config (**OpenAI-compatible**, **Ollama**, **Gemini**), v3 profile import, and Phase 1 tools.

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

### Run (Gemini, OpenAI-compatible API)

```bash
export OPENCLAUDE_PROVIDER=gemini
export GEMINI_API_KEY=...   # or GOOGLE_API_KEY
./openclaude
```

**CLI:** `./openclaude version`, `./openclaude doctor`, `./openclaude --help`  
**Flags:** `--config`, `--provider`, `--model`, `--base-url`  
**In-session:** `/help`, `/provider`, `/clear`, `/exit`

**Config layers:** v3 `.openclaude-profile.json` (cwd, then `$HOME`), then `openclaude.yaml` — see [docs/CONFIG.md](./docs/CONFIG.md) and [openclaude.example.yaml](./openclaude.example.yaml).

**Tools:** `FileRead`, `FileWrite`, `FileEdit`, `Bash`, `Grep`, `Glob`, `WebSearch`. Workspace = process working directory (paths cannot escape it). Details: [docs/SECURITY.md](./docs/SECURITY.md).

**Dangerous tools** prompt on stderr unless `OPENCLAUDE_AUTO_APPROVE_TOOLS=1` or `true`.

## Documents

| Doc | Purpose |
|-----|---------|
| [docs/CONFIG.md](./docs/CONFIG.md) | Env, YAML, flags, v3 migration hints |
| [docs/SECURITY.md](./docs/SECURITY.md) | Workspace boundary, dangerous tools, caveats |
| [docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md](./docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md) | What v3 does today (baseline) |
| [docs/DESIGN.md](./docs/DESIGN.md) | Target architecture and principles for v4 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased delivery plan |
| [TODO.md](./TODO.md) | Actionable checklist |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Build, test, lint, PR notes |
| [steps/step1.md](./steps/step1.md) | Go bootstrap notes + status |
| [steps/step2.md](./steps/step2.md) | Phase 1 tools + agent loop notes + status |

## Status

**Phase 2 (breadth slice done)** — v3 `.openclaude-profile.json` merge, `openclaude.yaml`, **`openai` / `ollama` / `gemini`**, explicit **codex** “not implemented” error, `doctor`, [docs/CONFIG.md](./docs/CONFIG.md). Tests: `internal/core/agent_test.go`, `internal/providers/openaicomp/client_test.go`, config profile tests.

**Phase 1** — Streaming tool loop — see [steps/step2.md](./steps/step2.md).

## License

TBD (align with v3 / project policy when code lands).
