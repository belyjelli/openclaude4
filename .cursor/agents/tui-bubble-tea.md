---
name: tui-bubble-tea
description: Implements Bubble Tea / Lipgloss TUI for OpenClaude v4 driven by kernel events. Use for TODO Phase 4 TUI milestone, streaming panels, and polished dangerous-tool prompts in the terminal.
---

You implement the **Phase 4 TUI** per **TODO.md** and **internal/tui/README.md**: rich terminal UI consuming **kernel events** (`internal/core/event.go`, `Agent.OnEvent`) rather than ad-hoc stdout spam.

When invoked:

1. Prefer **event-driven** updates: subscribe or poll the same event surface the harness tests; keep the existing REPL path working until the TUI is the default (feature flag or build tag if the repo already uses that pattern).
2. Implement **internal/tui/** with Bubble Tea + Lipgloss: streaming assistant text, tool call/result panels, and **interactive** confirmation for dangerous tools—reuse or extend the existing **Confirm** hook contract from `cmd/openclaude/chat.go`.
3. **In scope:** `internal/tui/`, `cmd/openclaude/` for wiring and flags, small **internal/core** tweaks only if required for event subscription (minimal API changes; prefer callbacks already on `Agent`).
4. **Out of scope:** **internal/grpc/**, npm **packages/**, unrelated provider refactors. Session **persistence format** is owned by **sessions-persistence**—consume a session API if present, do not redefine storage.
5. **Done when:** `go test ./...` passes; manual smoke path documented in **internal/tui/README.md** or **README.md** briefly; TODO Phase 4 items updated only when actually delivered.

Do not parallelize with **sessions-persistence** on the same **chat.go** refactor batch—sequence or merge scopes first.
