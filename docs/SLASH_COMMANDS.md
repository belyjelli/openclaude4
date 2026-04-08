# Slash commands: openclaude4 vs openclaude3

This compares **in-session** `/…` commands in:

- **openclaude4** — Go CLI (`openclaude` / `openclaude --tui`): router in [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go), help text in [`cmd/openclaude/chat.go`](../cmd/openclaude/chat.go) (`printChatHelpTo`). Helpers: [`slash_swap.go`](../cmd/openclaude/slash_swap.go), [`slash_extra.go`](../cmd/openclaude/slash_extra.go), [`slash_provider_wizard.go`](../cmd/openclaude/slash_provider_wizard.go). Live client swaps: [`internal/chatlive/live.go`](../internal/chatlive/live.go).
- **openclaude3** — TypeScript/Bun CLI: built-ins from `COMMANDS()` in [`src/commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) (sibling repo). v3 also registers **dynamic** slash commands (skills dirs, plugins, bundled skills, optional workflows/MCP skills), which are not listed exhaustively here.

v4 has a **fixed set** of built-in local slash commands plus **dynamic** `/<skill>` when the name matches the loaded skills catalog (case-insensitive). v3 has **many** more built-ins plus extensions; availability can depend on **build feature flags**, **auth**, and **`isEnabled()`** per command.

## Side-by-side (rough parity)

| Area | openclaude4 | openclaude3 |
|------|-------------|-------------|
| Help | `/help` | `/help` |
| Exit | `/exit`, `/quit` | `/exit`, `/quit` (alias on exit command) |
| Onboarding / doctor | `/onboard`, `/setup`, **`/doctor`** | `/doctor`, GitHub onboarding, many config/UI commands |
| Provider | `/provider`, **`/provider <name>`**, `/provider wizard` (stdin REPL; **TUI: `$EDITOR` + guide**) | `/provider` (interactive wizard, etc.) |
| MCP | `/mcp list`, **`/mcp config`**, `/mcp doctor`, **`/mcp add`** (shell hint), `/mcp help` | `/mcp` (broader subcommands + UI) |
| Transcript | `/clear`, `/compact` | `/clear`, `/compact` |
| Session | `/session …`, **`/resume`** | `/session`, `/resume`, … (richer) |
| Skills | `/skills list`, `/skills read`, **`/<skill>`** | `/skills` + skill-backed `/…` from disk/plugins |
| Side question | **`/btw`** (one-shot, no main transcript) | `/btw` (local-jsx side question) |
| Model | **`/model`**, flags, config | `/model`, … |
| Context / cost | **`/context`**, **`/tokens`**, **`/cost`**, **`/usage`** | `/context`, `/cost`, `/usage`, … |
| Clipboard / chrome | **`/copy`**, **`/theme`** (TUI), **`/vim`** (stub) | `/copy`, `/theme`, `/vim`, … |
| **Most other v3 commands** | Shell: `openclaude …`, config file, MCP tools | `/config`, `/init`, `/review`, `/permissions`, … |

## openclaude4 — full list

| Command | Notes |
|---------|--------|
| `/help` | Print REPL help |
| `/exit`, `/quit` | Leave chat |
| `/onboard`, `/setup` | Short onboarding hints |
| `/doctor` | Same output as `openclaude doctor` |
| `/context`, `/tokens` | Message count, rough tokens ([`RoughTokenEstimate`](../internal/session/tokens.go)), compact keep + threshold |
| `/model` | No args: print current model. `/model <id>`: set model for active provider (`viper` + new client; [`LiveChat`](../internal/chatlive/live.go)). TUI: blocked while a turn is in progress |
| `/provider` | Show provider info |
| `/provider wizard` | Plain REPL: stdin wizard. **TUI:** opens [`WritableConfigPath`](../internal/config/mcp_configfile.go) in `$VISUAL` / `$EDITOR` / `vi` after printing YAML/env guide |
| `/provider show`, `/status` | Same as bare `/provider` |
| `/provider help` | Subcommand help |
| `/provider openai\|ollama\|gemini\|github` | Switch `provider.name`, validate, new stream client |
| `/mcp`, `/mcp list` | Connected MCP servers + tools (this process) |
| `/mcp config` | Config file entries only ([`PrintMCPConfigList`](../cmd/openclaude/mcp.go)) |
| `/mcp doctor` | Same as list + `openclaude mcp doctor` tip |
| `/mcp add` | Prints shell hint (`openclaude mcp add …`) |
| `/mcp help` | Subcommand help |
| `/clear` | Clear messages (+ save session if enabled) |
| `/compact` | Tail keep from config |
| `/session …` | show, list, save, load, new, running, ps (unchanged) |
| `/resume` | No args: list sessions. `/resume <id>`: load session |
| `/skills list`, `/skills read <name>` | Skills catalog |
| `/<skill>` | If [`Catalog.GetFold`](../internal/skills/skills.go) matches, print skill body (same as read) |
| `/btw <question>` | [`core.SideQuestion`](../internal/core/sidechat.go): isolated completion, **not** appended to main transcript. TUI: blocked while busy |
| `/cost`, `/usage` | Transcript stats; **no** dollar billing |
| `/copy` | Last assistant message → clipboard (`pbcopy` / `xclip` / `wl-copy`) or print excerpt |
| `/theme light\|dark\|auto` | **TUI only:** [`ApplyTheme`](../internal/tui/styles.go) + glamour profile |
| `/vim` | **TUI:** message that vim keybindings are not implemented |
| Unknown | Error unless matched as a skill name |

## openclaude3 — built-in names (`COMMANDS()`)

Alphabetical **primary** `name` values from modules wired in [`commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) `COMMANDS()` (exact strings users type after `/`). This set **excludes** dynamically loaded skills/plugins/workflows/MCP skills.

`add-dir`, `advisor`, `agents`, `branch`, `brief` (Kairos), `buddy` (buddy feature), `btw`, `chrome`, `clear`, `color`, `compact`, `config`, `context`, `copy`, `cost`, `desktop`, `diff`, `doctor`, `dream`, `effort`, `exit`, `export`, `extra-usage`, `fast`, `feedback`, `files`, `heapdump`, `help`, `hooks`, `ide`, `init`, `insights`, `install-github-app`, `install-slack-app`, `keybindings`, `login`, `logout`, `mcp`, `memory`, `mobile`, `model`, `onboard-github`, `output-style`, `passes`, `permissions`, `plan`, `plugin`, `pr-comments`, `privacy-settings`, `provider`, `rate-limit-options`, `reload-plugins`, `release-notes`, `remote-control` (bridge feature), `remote-env`, `rename`, `resume`, `review`, `rewind`, `sandbox`, `security-review`, `session`, `skills`, `stats`, `status`, `statusline`, `stickers`, `tag`, `tasks`, `terminal-setup`, `theme`, `think-back`, `thinkback-play`, `upgrade`, `usage`, `ultrareview`, `vim`

**Also merged into `getCommands(cwd)`** (not all are in the static list above): bundled skills, plugin skills, skill-dir commands, plugin commands, optional workflow commands, and optional dynamic skills. Some entries in `COMMANDS()` are **feature-gated** at build time — see `feature('…')` branches in `commands.ts`.

**Internal / Ant-only** built-ins live in `INTERNAL_ONLY_COMMANDS` in the same file.

## v4 non-goals (vs v3)

Login/logout, Claude.ai flows, `/review` / security-review pipelines, GitHub app install, remote bridge, voice, plugins, workflows, heapdump, and most Ink-heavy wizards are **not** targeted in v4. Use **`openclaude` subcommands**, **config**, and **MCP** instead.

## Maintenance

When adding or renaming a v4 slash command, update:

1. [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go) (and related `slash_*.go` if needed)
2. [`printChatHelpTo`](../cmd/openclaude/chat.go)
3. This document

v3 inventory should stay aligned with `src/commands.ts` and the `commands/*/index.ts` (or equivalent) `name` / `aliases` fields in the openclaude3 tree.
