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

**CLI:** `./openclaude version`, `./openclaude doctor`, **`./openclaude mcp list`** / **`mcp doctor`** / **`mcp add`**, **`./openclaude serve`** (gRPC `openclaude.v4.AgentService`; see [internal/grpc/README.md](./internal/grpc/README.md)), `./openclaude --help`  
**Flags:** `--config`, `--provider`, `--model`, `--base-url`, `--print` / `-p` (one-shot script/CI; final reply on stdout), `--tui`, `--session`, `--resume`, `--list-sessions`, `--no-session`  
**Serve:** `--listen` or **`OPENCLAUDE_GRPC_ADDR`** (default `:50051`)  
**In-session:** `/help`, `/provider` (+ `/provider wizard` in plain REPL), `/mcp list`, `/mcp doctor`, `/session …`, `/compact`, `/clear`, `/exit`  
**TUI:** `./openclaude --tui` or `OPENCLAUDE_TUI=1` — full-screen Bubble Tea UI (streaming transcript, tool call/result blocks, permission prompts). Kernel uses [`OnEvent`](./internal/core/event.go) only; model text is not duplicated to stdout.

### Install (release binaries)

Tagged releases (semver, e.g. `v0.1.0`) publish archives via [GoReleaser](./goreleaser.yml) to [GitHub Releases](https://github.com/gitlawb/openclaude4/releases). Download the archive for your OS/arch and verify `checksums.txt` when provided.

**Config layers:** v3 `.openclaude-profile.json` (cwd, then `$HOME`), then `openclaude.yaml` — see [docs/CONFIG.md](./docs/CONFIG.md) and [openclaude.example.yaml](./openclaude.example.yaml).

**Tools:** `FileRead`, `FileWrite`, `FileEdit`, `Bash`, `Grep`, `Glob`, `WebSearch` (DuckDuckGo), `WebFetch` (direct HTTP), optional **`SpiderScrape`** when the [spider-rs](https://github.com/spider-rs/spider) `spider` CLI is on `PATH` (`cargo install spider_cli`) for richer local scrape — **no Firecrawl** in v4, **`Task`** (nested sub-agent loop), plus optional **MCP** tools from `mcp.servers` in config (`mcp_<server>__<tool>`). Workspace = process working directory (paths cannot escape it). Details: [docs/SECURITY.md](./docs/SECURITY.md) and [docs/CONFIG.md](./docs/CONFIG.md#mcp-servers).

**Dangerous tools** prompt on stderr unless `OPENCLAUDE_AUTO_APPROVE_TOOLS=1` or `true`.

## Documents

| Doc | Purpose |
|-----|---------|
| [docs/CONFIG.md](./docs/CONFIG.md) | Env, YAML, flags, precedence, v3 profile merge |
| [docs/SLASH_COMMANDS.md](./docs/SLASH_COMMANDS.md) | REPL `/` commands: v4 Go vs v3 TypeScript |
| [docs/PROVIDERS.md](./docs/PROVIDERS.md) | OpenAI / Ollama / Gemini — APIs, env, defaults |
| [docs/adr/0001-go-tooling-and-config.md](./docs/adr/0001-go-tooling-and-config.md) | ADR: Go version, releases, v3 config compatibility |
| [docs/SECURITY.md](./docs/SECURITY.md) | Workspace boundary, dangerous tools, caveats |
| [docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md](./docs/OPENCLAUDE3_ARCHITECTURE_SUMMARY.md) | What v3 does today (baseline) |
| [docs/DESIGN.md](./docs/DESIGN.md) | Target architecture and principles for v4 |
| [docs/ROADMAP.md](./docs/ROADMAP.md) | Phased delivery plan |
| [docs/MIGRATION_V3.md](./docs/MIGRATION_V3.md) | Moving from v3 CLI — config, commands, gRPC |
| [docs/RELEASE_CHECKLIST.md](./docs/RELEASE_CHECKLIST.md) | Semver, GoReleaser, changelog, security pointer |
| [docs/PROTO_VERSIONING.md](./docs/PROTO_VERSIONING.md) | v3 vs v4 gRPC package and compatibility |
| [docs/VSCODE_EXTENSION.md](./docs/VSCODE_EXTENSION.md) | Future VS Code extension milestone (planning) |
| [TODO.md](./TODO.md) | Actionable checklist |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Build, test, lint, PR notes |
| [steps/step1.md](./steps/step1.md) | Go bootstrap notes + status |
| [steps/step2.md](./steps/step2.md) | Phase 1 tools + agent loop notes + status |

## Status

**Phase 4** — **TUI** ([`internal/tui`](./internal/tui/)): Bubble Tea + Lipgloss, streaming + tool panels + permission UI from kernel events; **`Task`** sub-agent streams through the same `OnEvent` path. **`--tui`** / **`OPENCLAUDE_TUI`**. Release install story: GoReleaser + GitHub Releases (see above).

**Phase 3 (complete for Go CLI roadmap)** — **MCP** stdio client ([`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)), **`Task`** sub-agent tool, slash router (`/compact`, `/mcp`, …). Code: `internal/mcpclient`, `internal/core/task_tool.go`, `cmd/openclaude/slash.go`.

**Phase 2 (breadth slice done)** — v3 `.openclaude-profile.json` merge, `openclaude.yaml`, **`openai` / `ollama` / `gemini`**, explicit **codex** “not implemented” error, `doctor`, [docs/CONFIG.md](./docs/CONFIG.md). Tests: `internal/core/agent_test.go`, `internal/providers/openaicomp/client_test.go`, config profile tests.

**Phase 1** — Streaming tool loop — see [steps/step2.md](./steps/step2.md).

## License

TBD (align with v3 / project policy when code lands).
