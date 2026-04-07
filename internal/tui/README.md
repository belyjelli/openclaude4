# `internal/tui` — terminal UI (Phase 4)

Bubble Tea + Lipgloss front-end that renders **only from kernel [`core.Event`](../core/event.go) values** via [`Agent.OnEvent`](../core/agent.go). The model stream is not scraped from stdout: [`Agent.Out`](../core/agent.go) is set to `io.Discard` in [`Run`](./run.go).

## Layout

- Scrollable transcript (user, assistant stream, tool call/result blocks, permission outcomes, errors).
- Permission prompts use an inline panel; **y** / **n** / **Esc** respond (or set `OPENCLAUDE_AUTO_APPROVE_TOOLS` as in the plain REPL).
- Nested **Task** tool runs forward the parent’s `OnEvent`, so sub-agent streaming and tools appear in the same log.

## Entry

`openclaude --tui` or `OPENCLAUDE_TUI=1` (see [`cmd/openclaude/chat.go`](../../cmd/openclaude/chat.go)).
