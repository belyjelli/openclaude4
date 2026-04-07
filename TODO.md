# OpenClaude v4 — TODO / Checklist

Track implementation progress. Aligns with [docs/ROADMAP.md](./docs/ROADMAP.md).

**Active codebase:** Go single binary — `go.mod`, [`cmd/openclaude`](./cmd/openclaude/), [`internal/`](./internal/). Bootstrap notes: [`steps/step1.md`](./steps/step1.md). [docs/DESIGN.md](./docs/DESIGN.md) still describes an optional **npm `packages/`** layout; that track is listed at the bottom for doc alignment only.

---

## Phase 0 — Foundation (Go)

- [x] Git repo + docs + layout (`cmd/`, `internal/core`, `providers`, `tools`, `config`, …)
- [x] Go module + reproducible deps (`go.mod` / `go.sum`); build: `go build -o openclaude ./cmd/openclaude`
- [x] CI workflow: `go mod verify`, build, `go vet`, `go test` — [`.github/workflows/go.yml`](./.github/workflows/go.yml)
- [x] Goreleaser scaffold — [`goreleaser.yml`](./goreleaser.yml)
- [x] Stricter static analysis in CI — [`golangci-lint`](./.golangci.yml) in [`.github/workflows/go.yml`](./.github/workflows/go.yml)
- [x] ADR: Go version, release/goreleaser strategy, v3 config path compatibility — [`docs/adr/0001-go-tooling-and-config.md`](./docs/adr/0001-go-tooling-and-config.md)
- [x] [`CONTRIBUTING.md`](./CONTRIBUTING.md) (build, test, lint, PR expectations)

## Phase 1 — Kernel vertical slice (Go)

- [x] Multi-turn agent loop with streaming assistant text — [`internal/core/agent.go`](./internal/core/agent.go)
- [x] OpenAI-compatible streaming client with tools — [`internal/providers/openaicomp`](./internal/providers/openaicomp/)
- [x] Tool registry + JSON schemas for the API — [`internal/tools`](./internal/tools/)
- [x] `read_file` / write / edit tools, `bash` (dangerous; confirm hook in REPL), `grep`, `glob`, `web_search` — [`NewDefaultRegistry`](./internal/tools/registry.go)
- [x] Stdin/stdout REPL — [`cmd/openclaude/chat.go`](./cmd/openclaude/chat.go)
- [x] Env: `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`; flags `--model`, `--base-url`; `OPENCLAUDE_AUTO_APPROVE_TOOLS` — [README](./README.md), [`internal/config`](./internal/config/config.go)
- [ ] Explicit kernel **event** union (text delta, tool call/result, errors) for transports — today output goes to `io.Writer` + SDK message types
- [x] Agent loop tests with **local httptest** SSE + real `go-openai` stream reader (no external APIs) — [`internal/core/agent_test.go`](./internal/core/agent_test.go)
- [ ] Deeper Phase 1 provider/env doc in `docs/` (beyond README)

## Phase 2 — Config & providers

- [x] Partial: Viper + env keys + optional config file discovery — [`internal/config`](./internal/config/) ([`load.go`](./internal/config/load.go))
- [x] Config load order + viper precedence documented; `config.Validate()` for `provider.name` — [`load.go`](./internal/config/load.go), [`validate.go`](./internal/config/validate.go), [`precedence_test.go`](./internal/config/precedence_test.go)
- [x] Document migration from v3 (`.openclaude-profile.json` merged; `settings.json` manual) — [`docs/CONFIG.md`](./docs/CONFIG.md)
- [x] Second provider: **Ollama** (OpenAI-compatible `/v1` chat) — see provider wiring in `cmd/openclaude`
- [x] `openclaude doctor` — [`cmd/openclaude/doctor.go`](./cmd/openclaude/doctor.go)
- [ ] Integration tests with HTTP mocking **per provider** (agent tests cover OpenAI-style SSE only)

## Phase 3 — Tools & MCP

- [x] `grep` tool (uses `rg` when available — see tool implementation)
- [x] `glob` tool
- [ ] Sub-agent or task primitive (simpler than v3 is OK)
- [ ] MCP: transport connect, capability negotiation, tool proxying (`steps/step3.md`, `steps/step4.md` are design drafts)
- [x] Basic permission hook: REPL confirms dangerous tools before run
- [x] Slash commands: `/help`, `/provider`, `/clear`, `/exit` — extend set (`/help` text lists current)

## Phase 4 — Terminal UI

- [ ] `internal/tui` Bubble Tea / Lipgloss consuming kernel events only ([`internal/tui/README.md`](./internal/tui/README.md))
- [ ] Rich streaming + tool call/result panels (vs plain stdout today)
- [ ] Interactive permission prompts polished for TUI
- [ ] Published `bin` / install story (goreleaser releases, semver)

## Phase 5 — Sessions & compaction

- [ ] Session persistence on disk
- [ ] Resume last session; session listing
- [ ] Compaction / summarize over token threshold
- [ ] Tests for recovery after interrupted tool run

## Phase 6 — Headless & release

- [ ] gRPC or HTTP API server sharing kernel ([`internal/grpc/README.md`](./internal/grpc/README.md))
- [ ] Proto versioning vs v3 `openclaude.proto`
- [ ] Release checklist: semver, changelog, security policy pointer
- [ ] Migration guide from v3 CLI
- [ ] VS Code extension plan (separate milestone)

## Security & quality (continuous)

- [x] Path traversal / workspace boundary tests — [`internal/tools/workspace_boundary_test.go`](./internal/tools/workspace_boundary_test.go), [`paths_test.go`](./internal/tools/paths_test.go); notes in [`docs/SECURITY.md`](./docs/SECURITY.md)
- [ ] Secret scanning / redaction in transcripts
- [ ] Rate limit and timeout defaults documented (bash/tool HTTP clients)
- [ ] Dependency update policy (Dependabot/Renovate for Go modules)

---

## Reference — TypeScript `packages/` workspace (DESIGN.md)

Not started in this repo; kept for alignment with [docs/DESIGN.md](./docs/DESIGN.md) if you add a parallel npm workspace later.

- [ ] Root `package.json` + lockfile policy; strict TypeScript; `packages/core` placeholder
- [ ] ESLint/Biome + Prettier (or equivalent)
- [ ] CI: install, lint, typecheck, test on PR for that workspace

---

_Update this file as tasks complete; prefer linking PRs in commit messages rather than here._
