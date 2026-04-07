# `internal/tui` — terminal UI (Phase 4)

Bubble Tea + Lipgloss front-end that renders **only from kernel [`core.Event`](../core/event.go) values** via [`Agent.OnEvent`](../core/agent.go). The model stream is not scraped from stdout: [`Agent.Out`](../core/agent.go) is set to `io.Discard` in [`Run`](./run.go).

## Layout

- Scrollable transcript (user, assistant stream, tool call/result blocks, permission outcomes, errors). **PgUp** / **PgDn**, **Home** / **End**, and the mouse wheel move the view; new output sticks to the bottom unless you scroll up.
- Status line: provider, model, and session id; while the model runs, the current **tool** name is shown when known.
- Finished assistant turns can be rendered as **markdown** ([glamour](https://github.com/charmbracelet/glamour)); set `OPENCLAUDE_TUI_MARKDOWN=0` for plain text. Tool stdout preview length: `OPENCLAUDE_TUI_TOOL_PREVIEW` (UTF-8 runes, default 4000). Diff-like tool output gets simple line coloring.
- Permission prompts use an inline panel; **y** / **n** / **Esc** respond (or set `OPENCLAUDE_AUTO_APPROVE_TOOLS` as in the plain REPL).
- Nested **Task** tool runs forward the parent’s `OnEvent`, so sub-agent streaming and tools appear in the same log.

## Entry

`openclaude --tui` or `OPENCLAUDE_TUI=1` (see [`cmd/openclaude/chat.go`](../../cmd/openclaude/chat.go)).
