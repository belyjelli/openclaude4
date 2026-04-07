# OpenClaude v3 — Architecture Summary (Reference)

This document captures how **openclaude3** is structured today. Use it as a functional and architectural baseline for the v4 rewrite.

## Product shape

- **Terminal-first coding agent CLI** (`openclaude` binary) with streaming output, tool calling, slash commands, sub-agents, MCP, and multiple LLM backends (OpenAI-compatible, Gemini, Codex, Ollama, etc.).
- **Optional VS Code extension** (`vscode-extension/openclaude-vscode/`) for launch integration and theming.
- **Headless gRPC API** (`src/proto/openclaude.proto`, `AgentService.Chat` bidirectional stream) for embedding the agent in other apps or automation.

## Technology stack

| Area | Choice |
|------|--------|
| Language | TypeScript (ESM) |
| Runtime | Node ≥ 20 |
| Build / test | Bun (`scripts/build.ts`, `bun test`) |
| CLI UI | React 19 + Ink (custom reconciler) |
| CLI parsing | Commander (`@commander-js/extra-typings`) |
| Bundling | Single `dist/cli.mjs` via `Bun.build`, with compile-time `feature()` flags and MACRO defines |
| Validation / schemas | Zod, AJV where needed |
| RPC | gRPC (`@grpc/grpc-js`, proto-loader) |
| MCP | `@modelcontextprotocol/sdk` |

## Layered architecture (conceptual)

1. **Bootstrap / entry**
   - `src/entrypoints/cli.tsx`: minimal startup, env polyfills, `--version` fast path, provider profile hydration, validation, then many **feature-gated** subcommands (MCP sidecars, daemon, bridge, background sessions, templates, worktree+tmux, etc.) before loading the full app.
   - `src/main.tsx`: large surface — registers Commander commands, wires analytics/feature flags, prefetch (MCP, models), plugins, skills, dialogs, and mounts the interactive Ink experience.

2. **Agent core**
   - **API adapters** (`src/services/api/`): Anthropic SDK paths, OpenAI shim, Codex shim, retries, usage, agent routing (`agentRouting.ts`), provider config.
   - **Tool execution** (`src/services/tools/`): orchestration (concurrent vs serial batches via `isConcurrencySafe`), streaming executor, hooks, summaries.
   - **Tool definitions** (`src/tools/**`): one folder per tool (prompt snippets, Zod input, `run` / permission hooks). Includes bash, file ops, grep/glob, agents, tasks, MCP, web search/fetch, skills, LSP, etc.
   - **Shared tool contract** (`src/Tool.ts`): `buildTool`, context types, progress types, permission integration.

3. **Cross-cutting services**
   - MCP client and registry (`src/services/mcp/`)
   - LSP integration (`src/services/lsp/`)
   - Conversation compaction (`src/services/compact/`)
   - OAuth / credentials / secure storage hooks
   - Policy limits, remote-managed settings, plugins, team memory sync (where enabled)
   - Telemetry / analytics (with open-build stripping or no-op paths)

4. **State & UX**
   - App state (`src/state/`), hooks, Ink components under `src/components/`
   - Slash commands and command registry (`src/commands.js` and related)

5. **Large utility surface**
   - Session/conversation recovery, context/token budgeting, skills, swarm/teammates, worktrees, vim mode, ripgrep integration, provider profiles (`.openclaude-profile.json`), xdg paths, etc.

6. **Build / distribution**
   - `featureFlags` in `scripts/build.ts` disable internal-only modules; stub modules substituted for missing implementations.
   - Published package exposes `bin/openclaude` → bundled `dist/cli.mjs`.

## Notable engineering traits

- **Monolithic bundle** with **dynamic imports** and **compile-time feature gates** to keep cold paths lean.
- **Tight coupling** between UI (`main.tsx`), provider auth, and tool loop — high cohesion for shipping speed, harder to test or reuse in isolation.
- **Scale**: thousands of TS files; agent behavior is correct but navigation and onboarding for contributors are heavy.

This summary is intentionally high-level; v3 source remains the detailed spec for edge cases and provider quirks.
