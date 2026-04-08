# Slash commands: openclaude4 vs openclaude3

This compares **in-session** `/…` commands in:

- **openclaude4** — Go CLI (`openclaude` / `openclaude --tui`): router in [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go), help text in [`cmd/openclaude/chat.go`](../cmd/openclaude/chat.go) (`printChatHelpTo`).
- **openclaude3** — TypeScript/Bun CLI: built-ins from `COMMANDS()` in [`src/commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) (sibling repo). v3 also registers **dynamic** slash commands (skills dirs, plugins, bundled skills, optional workflows/MCP skills), which are not listed exhaustively here.

v4 has a **small fixed set** of local slash commands. v3 has **many** built-ins plus extensions; availability can depend on **build feature flags**, **auth**, and **`isEnabled()`** per command.

## Side-by-side (rough parity)

| Area | openclaude4 | openclaude3 |
|------|-------------|-------------|
| Help | `/help` | `/help` |
| Exit | `/exit`, `/quit` | `/exit`, `/quit` (alias on exit command) |
| Onboarding / doctor hints | `/onboard`, `/setup` (short text; see CONFIG) | `/doctor`, GitHub onboarding, many config/UI commands — **not** the same as v4 `/onboard` |
| Provider | `/provider`, `/provider wizard` (plain REPL), `/provider show\|status`, `/provider help` | `/provider` (interactive wizard, etc.) |
| MCP | `/mcp list`, `/mcp doctor`, `/mcp help` | `/mcp` (broader subcommands + UI) |
| Transcript | `/clear`, `/compact` | `/clear`, `/compact` |
| Session | `/session …` (show, list, save, load, new, running/ps) | `/session`, `/resume`, … (richer) |
| Skills | `/skills list`, `/skills read <name>` | `/skills` (broader); plus skill-backed `/…` from disk/plugins |
| Side question | — | `/btw` (local-jsx “side question”) |
| Model | flags / config only | `/model`, … |
| **Most other v3 commands** | Use shell: `openclaude doctor`, `openclaude mcp …`, config file | `/config`, `/init`, `/review`, `/permissions`, … (see built-in list below) |

## openclaude4 — full list

Implemented in `handleSlashLine` and helpers in [`slash.go`](../cmd/openclaude/slash.go).

| Command | Notes |
|---------|--------|
| `/help` | Print REPL help |
| `/exit`, `/quit` | Leave chat |
| `/onboard`, `/setup` | Short onboarding hints (env/YAML themes) |
| `/provider` | Active provider, model, base URL, key hint |
| `/provider wizard` | Interactive setup (stdin; **plain REPL only**; TUI shows static note) |
| `/provider show`, `/provider status` | Same as bare `/provider` |
| `/provider help` | Subcommand help text |
| `/mcp` or `/mcp list` | Describe connected MCP servers and tools |
| `/mcp doctor` | Same as list + tip to run `openclaude mcp doctor` |
| `/mcp help` | MCP slash help |
| `/clear` | Clear in-memory messages (+ save session if enabled) |
| `/compact` | Drop older turns; keep system + tail (count from config) |
| `/session` or `/session show` | Active session path + message count (requires sessions enabled) |
| `/session list` | Saved session files |
| `/session save` | Force persist |
| `/session load <id>` | Switch session (saves current first) |
| `/session new <id>` | New empty session id (saves current first) |
| `/session running`, `/session ps` | Local running registry (works even if disk session off) |
| `/skills list` | Loaded `SKILL.md` entries |
| `/skills read <name>` | Print one skill body |

Anything else starting with `/` is rejected (`unknown command — try /help`).

## openclaude3 — built-in names (`COMMANDS()`)

Alphabetical **primary** `name` values from modules wired in [`commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) `COMMANDS()` (exact strings users type after `/`). This set **excludes** dynamically loaded skills/plugins/workflows/MCP skills.

`add-dir`, `advisor`, `agents`, `branch`, `brief` (Kairos), `buddy` (buddy feature), `btw`, `chrome`, `clear`, `color`, `compact`, `config`, `context`, `copy`, `cost`, `desktop`, `diff`, `doctor`, `dream`, `effort`, `exit`, `export`, `extra-usage`, `fast`, `feedback`, `files`, `heapdump`, `help`, `hooks`, `ide`, `init`, `insights`, `install-github-app`, `install-slack-app`, `keybindings`, `login`, `logout`, `mcp`, `memory`, `mobile`, `model`, `onboard-github`, `output-style`, `passes`, `permissions`, `plan`, `plugin`, `pr-comments`, `privacy-settings`, `provider`, `rate-limit-options`, `reload-plugins`, `release-notes`, `remote-control` (bridge feature), `remote-env`, `rename`, `resume`, `review`, `rewind`, `sandbox`, `security-review`, `session`, `skills`, `stats`, `status`, `statusline`, `stickers`, `tag`, `tasks`, `terminal-setup`, `theme`, `think-back`, `thinkback-play`, `upgrade`, `usage`, `ultrareview`, `vim`

**Also merged into `getCommands(cwd)`** (not all are in the static list above): bundled skills, plugin skills, skill-dir commands, plugin commands, optional workflow commands, and optional dynamic skills. Some entries in `COMMANDS()` are **feature-gated** at build time (e.g. assistant, voice, fork, peers, torch, web-setup, workflows) — see `feature('…')` branches in `commands.ts`.

**Internal / Ant-only** built-ins (typical OSS builds omit these) live in `INTERNAL_ONLY_COMMANDS` in the same file — e.g. `commit`, `version`, `reset-limits`, `share`, `summary`, and other stubs or tooling commands.

## Gaps and how v4 can close them

v3 is a **full IDE-adjacent product** (Ink UI, plugins, auth, GitHub, remote bridge, many local-jsx flows). v4 is a **slim agent kernel** (Go, tools, MCP, sessions). **Full slash parity is not a realistic goal**; prioritize gaps that match v4’s scope.

### 1. High impact / fits v4 today

| Gap | Why it matters | How to close in v4 |
|-----|----------------|---------------------|
| **No `/model` (or `/provider` switch)** | Users must restart or edit YAML to change model | Add `/model <name>` (and optionally `/provider`) that updates viper + rebuilds or reconfigures `StreamClient` for the next turn; reject mid-stream if `busy`. Document which changes need restart. |
| **No in-chat `/doctor`** | Discoverability; v3 users expect it in the REPL | Delegate: `/doctor` runs the same checks as `openclaude doctor` and prints to the REPL/TUI transcript (spawn self or call shared diagnostic package). |
| **TUI `/provider wizard` is a stub** | Plain REPL has wizard; TUI does not | Port wizard to Bubble Tea (minimal steps) or open `$EDITOR` on the resolved config path with a generated snippet. |
| **No `/btw` (side question)** | v3 runs a parallel “side” completion without abandoning the main turn | Harder: needs a **second agent context** (isolated `[]Message` or single user+assistant pair), optional queue, and TUI state (`busy` vs side-rail). MVP: `/btw` as a **one-shot** extra completion that does not touch the main transcript, with results appended to transcript as a labeled block. |
| **Token / context visibility** | v3 `/context`; v4 already has [`RoughTokenEstimate`](../internal/session/tokens.go) | Add `/context` or `/tokens` printing rough token count, threshold, and message count (no fancy grid required at first). |

### 2. Medium value (more work or product decisions)

| Gap | Notes |
|-----|--------|
| **`/resume` in session** | v4 has `--resume`, `--list-sessions`; a slash command could list IDs and call the same `SwitchTo` path as `/session load`. |
| **`/mcp` subcommands** | Align with `openclaude mcp` (add, etc.) from REPL where safe; avoid mutating config without confirmation. |
| **Skill-backed `/foo` names** | v3 exposes many skills as slash commands; v4 only has `/skills list` + tools. Optional: register `name` from skill frontmatter into the slash router (collision rules, lazy load). |
| **`/cost`, `/usage`** | v4 does not track billing; would need usage headers from APIs or explicit non-goals. |
| **`/copy`, `/theme`, `/vim`, TUI chrome** | Quality-of-life; map to Bubble Tea settings and OS clipboard where available. |

### 3. Intentional non-goals (or very long horizon)

Login/logout, Claude.ai subscription flows, `/review` / `/security-review` pipelines, `/install-github-app`, remote bridge, voice, plugins, workflows, heapdump, and most **local-jsx**-style wizards are **out of scope** unless v4 explicitly expands into that product surface. Prefer **shell commands** (`openclaude …`), **config file**, and **MCP tools** instead.

### Implementation pattern for new v4 slashes

1. Add a `case` in [`slash.go`](../cmd/openclaude/slash.go) (or a small subpackage if the router grows).
2. For TUI, the existing `Config.Slash` callback already routes lines starting with `/`; return `appendOut` for transcript feedback.
3. Update [`printChatHelpTo`](../cmd/openclaude/chat.go) and this document.

## Maintenance

When adding or renaming a v4 slash command, update:

1. [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go)
2. [`printChatHelpTo` in `cmd/openclaude/chat.go`](../cmd/openclaude/chat.go)
3. This document

v3 inventory should stay aligned with `src/commands.ts` and the `commands/*/index.ts` (or equivalent) `name` / `aliases` fields in the openclaude3 tree.
