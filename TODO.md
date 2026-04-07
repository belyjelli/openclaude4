# OpenClaude v4 — TODO / Checklist

Track implementation progress. Aligns with [docs/ROADMAP.md](./docs/ROADMAP.md).

## Phase 0 — Foundation

- [ ] Initialize git repo (first commit: docs + workspace scaffold)
- [ ] Choose package manager and workspace tool; add root `package.json` + lockfile policy
- [ ] Add TypeScript base config (strict, `noImplicitAny`, path aliases if needed)
- [ ] Create `packages/core` (empty public API placeholder + README)
- [ ] Add lint + format (ESLint/Biome + Prettier or equivalent)
- [ ] Add CI workflow: install, lint, typecheck, test on PR
- [ ] Write ADR: Node version, bundler, compatibility with v3 config paths
- [ ] Add `CONTRIBUTING.md` (build, test, PR expectations)

## Phase 1 — Kernel vertical slice

- [ ] Define `Message`, `ToolCall`, `ToolResult`, and kernel `Event` types in `core`
- [ ] Define `Provider` interface: `streamCompletion` / `complete` with abort signal
- [ ] Implement OpenAI-compatible HTTP client (streaming SSE or fetch streams)
- [ ] Implement tool registry: register, list JSON schemas for API, dispatch by name
- [ ] Implement `read_file` tool (path validation, size limits, encoding)
- [ ] Implement `write_file` or `apply_patch` tool (choose one strategy for v4-first)
- [ ] Implement `bash` tool (cwd, timeout, stdout/stderr capture; danger flag documented)
- [ ] Wire agent loop: model → tool calls → results → model until done
- [ ] Add mock provider for unit tests (deterministic tool-call fixtures)
- [ ] Add stdin/stdout or minimal REPL transport for manual testing
- [ ] Document env vars for Phase 1 provider in `docs/` or package README

## Phase 2 — Config & providers

- [ ] Config schema (Zod or JSON Schema) + load order: file → env → flags
- [ ] Document migration notes from v3 `.openclaude-profile.json` / `settings.json`
- [ ] Implement second provider (e.g. Ollama or Gemini) behind same interface
- [ ] Add `doctor` or `openclaude doctor` subcommand (basics)
- [ ] Integration tests with HTTP mocking for each provider

## Phase 3 — Tools & MCP

- [ ] `grep` tool (prefer `rg` binary; document dependency)
- [ ] `glob` tool
- [ ] Sub-agent or task primitive (even if simpler than v3)
- [ ] MCP: transport connect, capability negotiation, tool proxying
- [ ] Permission model: per-tool policy + user prompt hook from transport
- [ ] Slash command parser + built-in commands (`/help`, `/clear`, …)

## Phase 4 — Terminal UI

- [ ] Create `packages/transport-cli` consuming kernel events only
- [ ] Render streaming assistant text; tool call / result panels
- [ ] Interactive permission prompts
- [ ] Binary entrypoint and `npm publish` `bin` map (can be pre-release)

## Phase 5 — Sessions & compaction

- [ ] Session persistence format on disk
- [ ] Resume last session; session listing
- [ ] Compaction / summarize when over token threshold
- [ ] Tests for recovery after interrupted tool run

## Phase 6 — Headless & release

- [ ] gRPC or HTTP API server package sharing kernel
- [ ] Decide proto versioning vs v3 `openclaude.proto`
- [ ] Release checklist: semver, changelog, security policy pointer
- [ ] Migration guide from v3 CLI for users
- [ ] VS Code extension plan (separate milestone issue list)

## Security & quality (continuous)

- [ ] Path traversal tests for all file tools
- [ ] Secret scanning / redaction in transcripts
- [ ] Rate limit and timeout defaults documented
- [ ] Dependency update policy (Renovate/Dependabot)

---

_Update this file as tasks complete; prefer linking PRs in commit messages rather than here._
