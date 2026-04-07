# Migrating from OpenClaude v3 CLI

OpenClaude **v4** (this repository) is a **single Go binary** with a smaller surface area than **v3** ([openclaude](https://github.com/Gitlawb/openclaude), TypeScript/Bun). This guide summarizes practical differences for users moving between them.

## Install and binary

- **v4:** `go build -o openclaude ./cmd/openclaude` or install from GitHub Releases for this repo. One process, no Node/Bun runtime.
- **v3:** Bun-based CLI and tooling as documented in the v3 README.

## Commands

- **v4 today:** default **chat** (stdin REPL or `--tui`), `openclaude version`, `openclaude doctor`. Session flags and `/session` are documented in [CONFIG.md](./CONFIG.md) and the README.
- **v4 headless API:** gRPC service code lives under [`internal/grpc/`](../internal/grpc/README.md). There is **no** `openclaude serve` subcommand yet; wiring is tracked in [TODO.md](../TODO.md) Phase 6.

## Configuration

Full precedence and file paths are in [CONFIG.md](./CONFIG.md). In short:

- v4 loads `./openclaude.{yaml,yml,json}` and `~/.config/openclaude/` (and related XDG paths), plus flags and environment variables.
- v3 **`.openclaude-profile.json`** is still **merged** into v4 config with **lower** precedence than v4 YAML — paths and merge order are documented in CONFIG.md.
- v3 **`settings.json` is not read by v4.** Relevant options must be translated manually to YAML or env (for example provider name, model, MCP servers).

## Providers

v4 supports **`openai`**, **`ollama`**, and **`gemini`** as documented in [PROVIDERS.md](./PROVIDERS.md). The **`codex`** provider name is recognized but returns a clear “not implemented” error until parity work lands (see [TODO.md](../TODO.md) “Gaps vs OpenClaude v3”).

## gRPC and clients

v3 exposes a bidi **`AgentService.Chat`** with `package openclaude.v1`. v4 uses **`openclaude.v4`** and different message names in places; existing v3 gRPC clients are **not** drop-in compatible. See [PROTO_VERSIONING.md](./PROTO_VERSIONING.md) and [`internal/grpc/README.md`](../internal/grpc/README.md) for mapping and versioning policy.

## UX parity

v4 offers slash commands, MCP via config, optional TUI, and on-disk sessions (Phase 5). Features that remain v3-only or partial in v4 are listed under **Gaps vs OpenClaude v3** in [TODO.md](../TODO.md).
