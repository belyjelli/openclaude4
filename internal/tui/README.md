# `internal/tui` — terminal UI (Phase 4)

Bubble Tea + Lipgloss front-end that renders **only from kernel [`core.Event`](../core/event.go) values** via [`Agent.OnEvent`](../core/agent.go). The model stream is not scraped from stdout: [`Agent.Out`](../core/agent.go) is set to `io.Discard` in [`Run`](./run.go).

## Layout

- Scrollable transcript (user, assistant stream, tool call/result blocks, permission outcomes, errors). **PgUp** / **PgDn**, **Home** / **End**, and the mouse wheel move the view; new output sticks to the bottom unless you scroll up.
- Status line: provider, model, and session id; while the model runs, the current **tool** name is shown when known.
- **Prompt chrome:** full-width horizontal rules above and below the **rich input row** (v3-style): a rounded **lipgloss** panel with subtle background, the **`❯`** prompt character (Ink `figures.pointer` in v3), optional **vim** insert/normal hint when `/vim` is on, then the text field. Under the lower rule, a **footer row** shows permission-style text on the left (auto-approve vs prompt for each dangerous tool) and **auto-compact** headroom on the right (`N% until auto-compact` when `session.compact_token_threshold` is set positive, else `auto-compact off`). **Shift+Tab** toggles runtime auto-approve for this session (same effect as `OPENCLAUDE_AUTO_APPROVE_TOOLS` at startup); a dim line is appended to the transcript when you toggle. Non-`ask` MCP server approval modes are summarized on the left when present (see `/permissions`). The placeholder line lists **Enter**, **Shift+Tab**, scrolling keys, and **`/help`**.
- Resizing the terminal sends `WindowSizeMsg`; viewport height, input width, rules, and the footer row reflow from the current `width`/`height` so the prompt stack stays aligned without an extra gap above the input.
- Finished assistant turns can be rendered as **markdown** ([glamour](https://github.com/charmbracelet/glamour)); set `OPENCLAUDE_TUI_MARKDOWN=0` for plain text. Tool stdout preview length: `OPENCLAUDE_TUI_TOOL_PREVIEW` (UTF-8 runes, default 4000). Diff-like tool output gets simple line coloring.
- Permission prompts use an inline panel; **y** / **n** / **Esc** respond. With auto-approve on (env or Shift+Tab), dangerous tools and MCP tools with `approval: ask` do not block on the panel.
- **Slash typeahead:** when the prompt starts with `/` (and the session is not busy), a suggestion block appears above the prompt rules with up to four matching commands (built-ins plus loaded skill names). **Tab** completes the first token to the selected row; **Shift+Tab** moves selection backward; **Up** / **Down** move selection when there is more than one match; **Esc** hides the overlay until you change the input. **Shift+Tab** still toggles auto-approve when the overlay is hidden.
- **Toast strip:** a single line under the status block and above the transcript shows errors, refusals, permission denials, and similar alerts; it auto-clears after about five seconds (new toasts replace the previous one).
- Nested **Task** tool runs forward the parent’s `OnEvent`, so sub-agent streaming and tools appear in the same log.

## Entry

`openclaude --tui` or `OPENCLAUDE_TUI=1` (see [`cmd/openclaude/chat.go`](../../cmd/openclaude/chat.go)).
