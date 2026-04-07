# OpenClaude v4

Greenfield rewrite of [OpenClaude](https://github.com/Gitlawb/openclaude) (v3 lives in **openclaude3**). This repository holds design docs and an **early Go CLI** (Phase 0).

## Build (Go)

Requires Go 1.22+ (see `go.mod`).

```bash
go build -ldflags "-X main.version=0.1.0 -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o openclaude ./cmd/openclaude
```

Run chat (needs `OPENAI_API_KEY`):

```bash
./openclaude
```

Use `./openclaude version` for build metadata. In-session commands: `/help`, `/provider`, `/clear`, `/exit`.

Optional environment: `OPENAI_BASE_URL`, `OPENAI_MODEL`. Flags: `--model`, `--base-url`.

## Documents

| Doc | Purpose |
|-----|---------|
| [docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md](./docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md) | What v3 does today (baseline) |
| [docs/DESIGN.md](./docs/DESIGN.md) | Target architecture and principles for v4 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased delivery plan |
| [TODO.md](./TODO.md) | Actionable checklist (TS-oriented; Go track follows `steps/`) |
| [steps/step1.md](./steps/step1.md) | Go bootstrap notes + status |

## Status

**Phase 0 (Go)** — Cobra CLI, OpenAI-compatible streaming chat, `/provider` REPL command. No tools or TUI yet. See [steps/step1.md](./steps/step1.md).

## License

TBD (align with v3 / project policy when code lands).
