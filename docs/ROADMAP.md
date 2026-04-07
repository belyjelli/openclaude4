# OpenClaude v4 — Roadmap

Phases are **sequential**; within a phase, some work can run in parallel (e.g. provider adapter + tool stubs).

---

## Phase 0 — Foundation (weeks 1–2)

**Goal:** Empty repo becomes a buildable monorepo with standards, not a feature-complete agent.

- Initialize workspace (TypeScript, strict mode, shared ESLint/Prettier or Biome, CI skeleton).
- Agree package layout (`packages/core`, `packages/providers`, …) per [DESIGN.md](./DESIGN.md).
- Add **architecture decision records** (ADRs) for: package manager, bundler, Node version, v3 compatibility policy.
- **No** user-facing CLI promise yet beyond ` —version` or a stub command.

**Exit criteria:** `pnpm install` / `npm ci` + lint + typecheck + unit test harness green in CI.

---

## Phase 1 — Agent kernel vertical slice (weeks 2–4)

**Goal:** One provider, minimal tools, no fancy TUI — prove the event-driven loop.

- Define **message types**, **tool contract**, and **kernel event** union in `core`.
- Implement **OpenAI-compatible** streaming completion (single provider first).
- Implement tools: **read file**, **write file** (or patch), **bash** (with cwd + basic allowlist or “yolo” dev mode clearly flagged).
- Implement **in-memory session** only; optional JSONL transcript log for debugging.
- Minimal **REPL or stdin/stdout** transport (no Ink required).

**Exit criteria:** From a shell, you can run a task that reads a file and runs a command with streamed assistant output; tests cover kernel without network (mock provider).

---

## Phase 2 — Configuration & provider breadth (weeks 4–7)

**Goal:** Real users can point v4 at their stack without recompiling.

- Config file schema + env overlay (document mapping from v3 env vars where relevant).
- Additional providers behind interfaces: **Gemini**, **Ollama**, **Codex** (order by demand).
- **Provider profiles** or documented migration from `.openclaude-profile.json`.
- **Doctor**-style command: Node version, `rg` presence, provider reachability (optional).

**Exit criteria:** Documented setup for at least OpenAI-compatible + one local path; CI integration tests with mocked HTTP.

---

## Phase 3 — Tool parity & MCP (weeks 7–12)

**Goal:** Approach v3’s day-to-day coding utility.

- Port or reimplement: **grep**, **glob**, **edit** strategies, **tasks** / **sub-agents** (simplified model first).
- **MCP client**: connect, list tools, invoke; permission model for MCP tool calls.
- **Slash commands** registry (subset matching v3 UX).

**Exit criteria:** Milestone checklist in [TODO.md](../TODO.md) marked for “MCP + file/shell parity”; dogfood on real repos.

---

## Phase 4 — Terminal UI & polish (weeks 12–16)

**Goal:** Replace stdin transport with a productive interactive CLI.

- Ink/React (or chosen TUI) **only** as `transport-cli`; consume kernel events.
- Streaming markdown / diff display, permission prompts, spinner/progress for tools.
- Optional: **vim** keybindings and **theme** hooks (lower priority than correctness).

**Exit criteria:** Interactive session feels comparable to v3 for common flows; no business logic in UI components.

---

## Phase 5 — Persistence, compaction, resume (weeks 16–20)

**Goal:** Long sessions remain usable.

- On-disk session format; **resume**; crash recovery basics.
- **Compaction** / summarization policy (can start simpler than v3).

**Exit criteria:** Resume after restart; token budget tests; documented limits.

---

## Phase 6 — Headless & ecosystem (weeks 20+)

**Goal:** Integrations and packaging.

- **gRPC** or HTTP SSE server sharing the same kernel (proto compatibility decision from Phase 0).
- **npm publish** pipeline, `bin` entry, install docs.
- **VS Code extension**: new repo or workspace package, deferred until CLI stable.

**Exit criteria:** Release candidate with semver, migration guide from v3.

---

## Ongoing (all phases)

- Security review for bash/file tools; path sandbox; secret redaction in logs.
- Performance: cold start, large repo grep; profiling hooks in kernel.
- Contributor docs: how to add a tool, how to add a provider.
