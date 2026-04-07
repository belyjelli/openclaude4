# VS Code extension (milestone plan)

The v3 codebase ships an editor integration under [`vscode-extension/openclaude-vscode`](https://github.com/Gitlawb/openclaude/tree/main/vscode-extension/openclaude-vscode). **v4 does not include a VS Code extension yet.** This document is a lightweight milestone plan for a future effort.

## Prerequisites

- **Stable CLI:** predictable config, `doctor`-style diagnostics, and documented flags (see [CONFIG.md](./CONFIG.md)).
- **Optional headless API:** when `openclaude serve` (or an HTTP equivalent) exists, an extension can attach to a long-lived process instead of only wrapping the REPL. See [ROADMAP.md](./ROADMAP.md) Phase 6 and [`internal/grpc/README.md`](../internal/grpc/README.md).

## Repository shape

- **Separate repository** vs **monorepo package** is an open product decision; [ROADMAP.md](./ROADMAP.md) already mentions either a new repo or a workspace package.

## Integration sketch (future)

- Spawn or attach to `openclaude` with workspace root = the editor’s workspace folder.
- Surface configuration and connectivity issues using checks analogous to `doctor`.
- Thin UI over stdin REPL or, later, over the gRPC/HTTP API for streaming and tool visibility.

No implementation commitment is implied; treat this file as planning notes only.
