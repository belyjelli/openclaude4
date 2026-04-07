# OpenClaude v4 — Design Outline

## Executive summary

**OpenClaude v4** is a **from-scratch implementation** of the same product intent as v3: a terminal coding agent with pluggable model backends, rich tools (files, shell, MCP, web, sub-agents), and optional headless integration. The rewrite should **preserve user-visible capabilities** where they matter while **replacing the internal shape**: strict module boundaries, a small documented **core agent runtime**, test-first development, and a UI layer that consumes a stable **session / event API** instead of reaching into execution internals.

Success looks like: faster contributor onboarding, reliable CI, easier addition of providers and tools, and the option to run the same engine in CLI, gRPC, or tests without duplicating logic.

## Design principles

1. **Core vs adapters** — One “agent kernel” (message history, model round-trips, tool dispatch, permissions) with thin provider and transport adapters.
2. **Explicit boundaries** — Packages or top-level folders with **one-way dependencies** (e.g. `ui → session`, `session → tools`, `tools → fs/shell`, never the reverse).
3. **Events in, events out** — The kernel exposes a stream or iterator of **typed events** (text delta, tool call, tool result, permission prompt, error, done). UI and gRPC are subscribers.
4. **Tool plugin surface** — Tools register name, JSON Schema (or Zod→schema), concurrency policy, and executor; no direct UI imports inside executors.
5. **Configuration as data** — Providers, MCP servers, and feature toggles loaded from validated config files + env; no silent globals where avoidable.
6. **Compatibility strategy** — Document which v3 artifacts remain supported (e.g. profile file shape, settings paths) vs which are intentionally broken in v4.

## Target module map (suggested)

```
packages/
  core/           # Agent kernel: loop, messages, tool registry, permissions model
  providers/      # OpenAI-compatible, Gemini, … — implement shared Provider interface
  tools/          # Built-in tools only; optional workspace package for third-party tools later
  mcp/            # MCP client bridge used by core
  config/         # Schema, load/merge, env overlay
  transport-cli/  # Ink/React or lighter TUI — consumes core events
  transport-grpc/ # gRPC service mapping ↔ core events (optional phase)
  testing/        # Fakes, golden transcripts, provider mocks
```

A single-repo **npm/pnpm workspace** (or Bun workspaces) keeps this manageable without premature publishing.

## Core agent loop (logical)

1. Append user message; apply system prompt and context window policy.
2. Call `Provider.completeStream(messages, tools, options)` → stream of assistant content + tool_use blocks.
3. Partition tool calls (read-only concurrent batches vs serial mutating) — same semantic rule as v3 unless simplified deliberately.
4. For each tool: check permissions → execute → append tool results; repeat until model stops or limits hit.
5. Emit lifecycle events for UI/transports; persist session if enabled.

## What to carry forward from v3 (behavioral)

- Multi-provider support and **OpenAI-compatible** path as the default integration surface.
- Tool set parity **by milestone** (see ROADMAP): start with bash + read/write/edit + grep/glob, then MCP, then agents/tasks, then web, etc.
- Session resume / compaction concepts — implement after core loop is stable.
- gRPC proto **can** stay compatible (`openclaude.proto`) for integrators, or version as `v2` if breaking changes are required (document either way).

## What to change on purpose

- **Split** `main.tsx`-style god modules into **composition roots** (small `main` that only wires config + transport + kernel).
- **Replace** implicit global singletons with **injected** clock, fs, process spawn, and config.
- **Centralize** provider-specific quirks in provider modules, not scattered `if (provider)` in UI.
- **Testing**: contract tests per provider adapter; integration tests with recorded streams; smoke CLI tests in CI.

## Non-goals (initial v4 phases)

- Pixel-perfect replication of every v3 internal optimization or internal-only feature flag.
- Porting the entire VS Code extension before the CLI kernel is usable.
- Full telemetry parity with any closed/analytics pipeline unless explicitly in scope.

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Underestimating v3 edge cases | Keep v3 summary doc updated; milestone “parity checklist” per tool/provider |
| Second system syndrome | Ship a **thin vertical slice** (one provider, three tools, REPL) before breadth |
| Ink/React complexity | Consider minimal TUI first, or isolate Ink behind `transport-cli` |

## Open decisions (resolve in Phase 0)

- Package manager and exact workspace layout (`pnpm` vs `npm` vs `bun`).
- Whether v4 remains **Bun-build** like v3 or uses **tsup/esbuild** for wider tooling familiarity.
- Minimum Node version (20 LTS vs 22).
- License and publishing identity (`@gitlawb/openclaude` vs new scope).
