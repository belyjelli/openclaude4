# OpenClaude v4 — TODO / Checklist

Track implementation progress. Aligns with [docs/ROADMAP.md](./docs/ROADMAP.md).

**Active codebase:** Go single binary — `go.mod`, [`cmd/openclaude`](./cmd/openclaude/), [`internal/`](./internal/). Bootstrap notes: [`steps/step1.md`](./steps/step1.md). [docs/DESIGN.md](./docs/DESIGN.md) still describes an optional **npm `packages/`** layout; that track is listed at the bottom for doc alignment only.

**OpenClaude v3 baseline:** The TypeScript repo (`openclaude3`) is the functional reference: Ink TUI, large tool surface (incl. skills, LSP, web fetch), many providers, plugins, agent routing, always-on session/transcript model, gRPC with `session_id`, VS Code extension, etc. **v4 does not yet match that breadth**; the checklist below plus **Gaps vs v3** track what is still missing or only partial.

---

## Stub backlog & doc follow-ups

- [x] **LICENSE** — root [LICENSE](./LICENSE) + [README.md](./README.md) link.
- [x] **Codex early validation** — [`internal/providererrs`](./internal/providererrs/codex.go) + [`config.Validate`](./internal/config/validate.go) returns `ErrCodexNotImplemented` for `provider.name: codex` (same sentinel as [`NewStreamClient`](./internal/providers/runtime.go)).
- [x] **`/vim` TUI** — vim-style prompt subset in Bubble Tea; [docs/SLASH_COMMANDS.md](./docs/SLASH_COMMANDS.md).
- [x] **gRPC multimodal** — `ChatRequest.image_url` / `image_inline` + [`internal/grpc/server.go`](./internal/grpc/server.go) → `RunUserTurnMulti`; [docs/gRPC_COMPATIBILITY.md](./docs/gRPC_COMPATIBILITY.md).

---

## Gaps vs OpenClaude v3 (parity backlog)

Unchecked items are **not** covered at v3 depth in v4 yet (even when a smaller alternative exists).

### Providers and auth

- [ ] **Codex** provider (v4 returns `ErrCodexNotImplemented` from [`config.Validate`](./internal/config/validate.go) and [`NewStreamClient`](./internal/providers/runtime.go))
- [x] **GitHub Models** provider — [`openaicomp.NewGitHubModels`](./internal/providers/openaicomp/github.go); `OPENCLAUDE_PROVIDER=github`, `GITHUB_TOKEN` / `GITHUB_PAT`; interactive wizard in [`cmd/openclaude/slash_provider_wizard.go`](./cmd/openclaude/slash_provider_wizard.go); see [docs/PROVIDERS.md](./docs/PROVIDERS.md) for setup
- [ ] **Atomic Chat**, **Bedrock / Vertex / Foundry** and other env-driven backends listed in [v3 README](https://github.com/Gitlawb/openclaude) “Supported Providers”
- [ ] Optional: v3-style **secure storage / keychain** hydration for Gemini and GitHub (beyond env + `.openclaude-profile.json` merge)

### Tools and agent behavior

- [x] **WebFetch** tool — [`internal/tools/web_fetch.go`](./internal/tools/web_fetch.go): HTTP(S) GET, HTML→text, SSRF-minded host/IP checks, caps documented in [SECURITY.md](./docs/SECURITY.md)
- [x] Optional **spider_cli** — when `spider` is on `PATH`, **[`SpiderScrape`](./internal/tools/spider_scrape.go)** is registered (single-URL scrape). **No Firecrawl** — v3’s `FIRECRAWL_API_KEY` path is intentionally omitted; use **SpiderScrape** for richer local scrape.
- [x] Partial: **Skills** — [`internal/skills`](./internal/skills/skills.go) loads `<dir>/<name>/SKILL.md` (+ YAML frontmatter); tools **SkillsList** / **SkillsRead**; [`/skills list|read`](./cmd/openclaude/slash.go); dirs: `skills.dirs`, `OPENCLAUDE_SKILLS_DIRS`, default `.openclaude/skills` and `~/.local/share/openclaude/skills` when present. **No** v3 plugin CLI yet.
- [x] Partial: **LSP-shaped Go outline** — **GoOutline** tool ([`internal/tools/go_outline.go`](./internal/tools/go_outline.go)) lists top-level declarations via `go/parser` (not a language server).
- [x] Partial: **Multimodal / vision** — [`core.RunUserTurnMulti`](./internal/core/agent.go) + [`core.BuildUserContentParts`](./internal/core/multipart.go); flags [`--image-url`](./cmd/openclaude/root.go) / [`--image-file`](./cmd/openclaude/root.go) (first user message in REPL/TUI; with `-p`). gRPC: [`ChatRequest.image_url` / `image_inline`](./internal/grpc/proto/openclaude.proto) on `openclaude serve`.
- [x] Partial: **Agent routing** — [`agent_routing.task_model`](./internal/config/agent_routing.go) / `OPENCLAUDE_AGENT_TASK_MODEL` selects model for **Task** sub-agent when the client is `*openaicomp.Client`. Full v3-style multi-agent routing still open.

### CLI / UX

- [x] **Interactive `/provider` wizard** — [`/provider wizard`](./cmd/openclaude/slash.go) + [`slash_provider_wizard.go`](./cmd/openclaude/slash_provider_wizard.go); TUI falls back to static copy-paste guide
- [x] **Headless one-shot** mode (v3 `-p` / print) for scripts and CI — [`runPrintTurn`](./cmd/openclaude/chat.go); `--print` / `-p` (optional `-p -` stdin); incompatible with `--tui`; dangerous tools need `OPENCLAUDE_AUTO_APPROVE_TOOLS` or they are skipped (stderr)
- [x] Optional: **concurrent session registry** / `ps`-style listing — [`<dir>/running/<pid>.json`](./internal/session/running.go) on chat/TUI start; [`openclaude sessions`](./cmd/openclaude/sessions.go); [`/session running`](./cmd/openclaude/slash.go) / `/session ps`
- [x] Partial: **slash commands** — [`/onboard`](./cmd/openclaude/slash.go) / `/setup`, [`/mcp help`](./cmd/openclaude/slash.go); v3-deep items still open (e.g. `/onboard-github`, full MCP config from REPL)

### MCP

- [x] **MCP subcommands** — [`openclaude mcp list`](./cmd/openclaude/mcp.go), [`mcp doctor`](./cmd/openclaude/mcp.go), [`mcp add`](./cmd/openclaude/mcp.go) (append to config via [`AppendMCPServerToConfigFile`](./internal/config/mcp_configfile.go)); REPL [`/mcp list`](./cmd/openclaude/slash.go) / [`/mcp doctor`](./cmd/openclaude/slash.go)

### Headless gRPC and extension

- [x] **CLI to start gRPC** — [`openclaude serve`](./cmd/openclaude/serve.go); kernel + tests under [`internal/grpc`](./internal/grpc/README.md)
- [x] **v3 proto parity documentation** — [`docs/gRPC_COMPATIBILITY.md`](./docs/gRPC_COMPATIBILITY.md) provides migration guide, wire-level differences table, and compatibility gateway pattern; v3 uses `openclaude.v1` with `ActionRequired` / `FinalResponse` / `ErrorResponse`; v4 uses `openclaude.v4` with different event names; `session_id` supported on v4 `ChatRequest` for on-disk sessions
- [ ] VS Code extension plan remains Phase 6 (v3 ships [`vscode-extension/openclaude-vscode`](https://github.com/Gitlawb/openclaude/tree/main/vscode-extension/openclaude-vscode)); planning doc: [docs/VSCODE_EXTENSION.md](./docs/VSCODE_EXTENSION.md)

### Engineering / correctness (session + CLI)

- [x] **`/session list` + `--list-sessions`** — use [`session.Entry`](./internal/session/list.go) fields consistently ([`slash.go`](./cmd/openclaude/slash.go), [`chat.go`](./cmd/openclaude/chat.go)); `--list-sessions` runs **before** provider validation so listing works without API keys
- [x] Align **[docs/CONFIG.md](./docs/CONFIG.md)** / [README](./README.md) session semantics with [`resolveChatPersistence`](./cmd/openclaude/chat.go) (default random id + save unless `--no-session`)

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
- [x] `read_file` / write / edit tools, `bash` (dangerous; confirm hook in REPL), `grep`, `glob`, `web_search`, `web_fetch`, optional `spider_scrape` when `spider` on PATH — [`NewDefaultRegistry`](./internal/tools/registry.go)
- [x] Stdin/stdout REPL — [`cmd/openclaude/chat.go`](./cmd/openclaude/chat.go)
- [x] Env: `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`; flags `--model`, `--base-url`; `OPENCLAUDE_AUTO_APPROVE_TOOLS` — [README](./README.md), [`internal/config`](./internal/config/config.go)
- [x] Kernel **event harness** — [`internal/core/event.go`](./internal/core/event.go) (`Event`, `EventKind`) + [`Agent.OnEvent`](./internal/core/agent.go); REPL still uses `Out` only
- [x] Agent loop tests with **local httptest** SSE + real `go-openai` stream reader (no external APIs) — [`internal/core/agent_test.go`](./internal/core/agent_test.go) (includes OpenAI / Ollama / Gemini **model id** cases on the same HTTP shape)
- [x] Provider overview in [`docs/PROVIDERS.md`](./docs/PROVIDERS.md) (cross-linked from [CONFIG.md](./docs/CONFIG.md))

## Phase 2 — Config & providers

- [x] Partial: Viper + env keys + optional config file discovery — [`internal/config`](./internal/config/) ([`load.go`](./internal/config/load.go))
- [x] Config load order + viper precedence documented; `config.Validate()` for `provider.name` — [`load.go`](./internal/config/load.go), [`validate.go`](./internal/config/validate.go), [`precedence_test.go`](./internal/config/precedence_test.go)
- [x] Document migration from v3 (`.openclaude-profile.json` merged; `settings.json` manual) — [`docs/CONFIG.md`](./docs/CONFIG.md)
- [x] Second provider: **Ollama** (OpenAI-compatible `/v1` chat) — see provider wiring in `cmd/openclaude`
- [x] `openclaude doctor` — [`cmd/openclaude/doctor.go`](./cmd/openclaude/doctor.go)
- [x] HTTP-mocked agent tests for each **OpenAI-compatible** provider model id (same SSE; asserts `model` in JSON body) — [`internal/core/agent_test.go`](./internal/core/agent_test.go)

## Phase 3 — Tools & MCP

- [x] `grep` tool (uses `rg` when available — see tool implementation)
- [x] `glob` tool
- [x] **Task** tool — bounded sub-session, fresh system + user goal, same tools/client, stdout discarded for sub-run; child registry omits `Task` (no recursion) — [`internal/core/task_tool.go`](./internal/core/task_tool.go)
- [x] MCP: stdio `ConnectAndRegister`, tool list + `CallTool` proxy, YAML `mcp.servers`, `/mcp list`, `doctor` prints configured servers, **`openclaude mcp list` / `mcp doctor`** — [`internal/mcpclient`](./internal/mcpclient/), [`internal/config/mcp.go`](./internal/config/mcp.go), [`cmd/openclaude/mcp.go`](./cmd/openclaude/mcp.go)
- [x] Basic permission hook: REPL confirms dangerous tools before run
- [x] Slash commands — [`cmd/openclaude/slash.go`](./cmd/openclaude/slash.go): `/help`, `/provider`, `/mcp`, `/skills`, `/session` (when persistence enabled), `/compact`, `/clear`, `/exit`, `/quit`

## Phase 4 — Terminal UI

- [x] `internal/tui` Bubble Tea / Lipgloss consuming kernel events only ([`internal/tui/README.md`](./internal/tui/README.md))
- [x] Rich streaming + tool call/result panels (vs plain stdout today)
- [x] Interactive permission prompts polished for TUI
- [x] Published `bin` / install story (goreleaser releases, semver)
- [x] TUI polish: status line (provider · model · session), **PgUp/PgDn/Home/End** transcript scroll with stick-to-bottom on new output ([`internal/tui/model.go`](./internal/tui/model.go)), **glamour** markdown on finished assistant turns ([`internal/tui/render.go`](./internal/tui/render.go); `OPENCLAUDE_TUI_MARKDOWN=0` to disable), diff-like **tool result** coloring, configurable tool preview (`OPENCLAUDE_TUI_TOOL_PREVIEW` rune cap, default 4000)

## Phase 5 — Sessions & compaction

- [x] Partial: on-disk session JSON + [`internal/session`](./internal/session/) (`Store`, `Handle` path, resume id, token helpers)
- [x] Partial: default **on-disk** session (random id per run unless `--session` / `OPENCLAUDE_SESSION`, or `--resume` / last-id file) with `--no-session` to disable — differs from v3 **path/layout** and **jsonl transcript** model; see **Gaps vs v3** for full parity
- [x] **`/session list` + `--list-sessions`** wired (see **Gaps vs v3** engineering notes)
- [x] **[`session.ApplyTokenThreshold`](./internal/session/tokens.go)** before each user turn in stdin REPL and TUI ([`BeforeUserTurn`](./internal/tui/model.go)) when `session.compact_token_threshold` is set positive; manual [`/compact`](./cmd/openclaude/slash.go) still uses lossy tail via [`session.CompactTail`](./internal/session/transcript.go)
- [x] Tests for recovery after interrupted tool run — [`internal/session/store_test.go`](./internal/session/store_test.go) (`TestRepairInterruptedToolRound`), [`internal/core/agent_test.go`](./internal/core/agent_test.go) (`TestRunUserTurn_RecoveredInterruptedToolTranscript`)

## Phase 6 — Headless & release

- [x] Partial: gRPC **server + generated stubs + v3 mapping table** — [`internal/grpc/README.md`](./internal/grpc/README.md)
- [x] **`openclaude serve`** (shared bootstrap with chat: config, client, registry, MCP, Task tool) — [`cmd/openclaude/serve.go`](./cmd/openclaude/serve.go)
- [x] Optional: **gRPC stream session binding** (`session_id` on [`ChatRequest`](./internal/grpc/proto/openclaude.proto), on-disk [`session.Store`](./internal/session/store.go) when sessions enabled)
- [x] Release checklist: semver, changelog, security policy pointer — [docs/RELEASE_CHECKLIST.md](./docs/RELEASE_CHECKLIST.md)
- [x] Migration guide from v3 CLI (config, flags, gRPC package / event names) — [docs/MIGRATION_V3.md](./docs/MIGRATION_V3.md), [docs/PROTO_VERSIONING.md](./docs/PROTO_VERSIONING.md)
- [x] VS Code extension plan (separate milestone) — [docs/VSCODE_EXTENSION.md](./docs/VSCODE_EXTENSION.md)

## Security & quality (continuous)

- [x] Path traversal / workspace boundary tests — [`internal/tools/workspace_boundary_test.go`](./internal/tools/workspace_boundary_test.go), [`paths_test.go`](./internal/tools/paths_test.go); notes in [`docs/SECURITY.md`](./docs/SECURITY.md)
- [x] Secret scanning / redaction in transcripts — [`internal/core/redact.go`](./internal/core/redact.go), [`docs/SECURITY.md`](./docs/SECURITY.md#transcript-and-log-redaction)
- [x] Rate limit and timeout defaults documented (bash/tool HTTP clients) — [`docs/SECURITY.md`](./docs/SECURITY.md#network), [`docs/CONFIG.md`](./docs/CONFIG.md#timeouts-iteration-limits-and-http-behavior)
- [x] Dependabot for Go modules — [`.github/dependabot.yml`](./.github/dependabot.yml)

---

## Reference — TypeScript `packages/` workspace (DESIGN.md)

Not started in this repo; kept for alignment with [docs/DESIGN.md](./docs/DESIGN.md) if you add a parallel npm workspace later.

- [ ] Root `package.json` + lockfile policy; strict TypeScript; `packages/core` placeholder
- [ ] ESLint/Biome + Prettier (or equivalent)
- [ ] CI: install, lint, typecheck, test on PR for that workspace

---

_Update this file as tasks complete; prefer linking PRs in commit messages rather than here._
