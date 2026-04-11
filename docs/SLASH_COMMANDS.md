# Slash commands: openclaude4 vs openclaude3

This compares **in-session** `/…` commands in:

- **openclaude4** — Go CLI (`openclaude` / `openclaude --tui`): router in [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go), help text in [`cmd/openclaude/chat.go`](../cmd/openclaude/chat.go) (`printChatHelpTo`). Helpers: [`slash_swap.go`](../cmd/openclaude/slash_swap.go), [`slash_extra.go`](../cmd/openclaude/slash_extra.go), [`slash_export.go`](../cmd/openclaude/slash_export.go), [`slash_provider_wizard.go`](../cmd/openclaude/slash_provider_wizard.go). Live client swaps: [`internal/chatlive/live.go`](../internal/chatlive/live.go).
- **openclaude3** — TypeScript/Bun CLI: built-ins from `COMMANDS()` in [`src/commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) (sibling repo). v3 also registers **dynamic** slash commands (skills dirs, plugins, bundled skills, optional workflows/MCP skills), which are not listed exhaustively here.

v4 has a **fixed set** of built-in local slash commands plus **dynamic** `/<skill>` when the name matches the loaded skills catalog (case-insensitive). v3 has **many** more built-ins plus extensions; availability can depend on **build feature flags**, **auth**, and **`isEnabled()`** per command.

In **`openclaude --tui`**, the prompt offers **slash typeahead**: after `/`, matching commands (including skill names) appear above the input with **Tab** to complete and arrow keys to choose—see [`internal/tui/README.md`](../internal/tui/README.md).

## Side-by-side (rough parity)

| Area | openclaude4 | openclaude3 |
|------|-------------|-------------|
| Help | `/help` | `/help` |
| Exit | `/exit`, `/quit` | `/exit`, `/quit` (alias on exit command) |
| Onboarding / doctor | `/onboard`, `/setup`, **`/doctor`** | `/doctor`, GitHub onboarding, many config/UI commands |
| Provider | `/provider`, **`/provider <name>`**, `/provider wizard` (step-by-step REPL or **TUI panel**) | `/provider` (interactive wizard, etc.) |
| MCP | `/mcp list`, **`/mcp config`**, `/mcp doctor`, **`/mcp add`** (shell hint), `/mcp help` | `/mcp` (broader subcommands + UI) |
| Transcript | `/clear`, `/compact` | `/clear`, `/compact` |
| Session | `/session …`, **`/resume`** | `/session`, `/resume`, … (richer) |
| Skills | `/skills list`, `/skills read`, **`/<skill>`** | `/skills` + skill-backed `/…` from disk/plugins |
| Side question | **`/btw`** (one-shot, no main transcript) | `/btw` (local-jsx side question) |
| Model | **`/model`**, flags, config | `/model`, … |
| Context / cost | **`/context`**, **`/tokens`**, **`/cost`**, **`/usage`** | `/context`, `/cost`, `/usage`, … |
| Clipboard / chrome | **`/copy`**, **`/theme`** (TUI), **`/vim`** (TUI: vim-style prompt subset) | `/copy`, `/theme`, `/vim`, … |
| **Most other v3 commands** | Shell: `openclaude …`, config file, MCP tools + **`/config`**, **`/export`**, **`/init`**, **`/permissions`**, **`/version`** | `/review`, … |

## openclaude4 — full list

| Command | Notes |
|---------|--------|
| `/help` | Print REPL help |
| `/exit`, `/quit` | Leave chat |
| `/onboard`, `/setup` | Short onboarding hints |
| `/doctor` | Same output as `openclaude doctor` |
| `/config` | [`DescribeEffectiveConfig`](../internal/config/describe.go): precedence, `viper` file, v3 profile path, search paths, writable config hint, provider/model/session/MCP names + approval (no secrets) |
| `/permissions` | `OPENCLAUDE_AUTO_APPROVE_TOOLS`, MCP `approval` per server, cwd, pointer to [SECURITY.md](./SECURITY.md) |
| `/version` | Same line as `openclaude version` (embedded version/commit) |
| `/init` | Starter `openclaude.yaml` snippet + pointers to `openclaude.example.yaml` and CONFIG.md |
| `/export` | In-memory transcript → JSON ([`session.FileV1`](../internal/session/file_v1.go)) or Markdown; `/export`, `/export json`, `/export md`, optional path; large stdout exports require a path ([`slash_export.go`](../cmd/openclaude/slash_export.go)) |
| `/context`, `/tokens` | Message count, rough tokens ([`RoughTokenEstimate`](../internal/session/tokens.go)), compact keep + threshold |
| `/model` | No args: print current model. `/model <id>`: set model for active provider (`viper` + new client; [`LiveChat`](../internal/chatlive/live.go)). TUI: blocked while a turn is in progress |
| `/provider` | Show provider info |
| `/provider wizard` | **REPL:** stdin prompts; type **`b`** / **`back`** to go back. **TUI:** in-app bordered panel ([`internal/tui/provider_wiz.go`](../internal/tui/provider_wiz.go)): **↑↓** move, **Enter** confirm, **`b`** back, **esc** cancel. Prints YAML/env snippet to the transcript when done (restart after editing config). Optional Ollama model list via host `/api/tags`. |
| `/provider show`, `/status` | Same as bare `/provider` |
| `/provider help` | Subcommand help |
| `/provider openai\|ollama\|gemini\|github\|openrouter` | Switch `provider.name`, validate, new stream client |
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
| `/theme light\|dark\|auto` | **TUI only:** [`ApplyTheme`](../internal/tui/styles.go) + Chroma style profile (light vs dark) for assistant markdown |
| `/vim` | **TUI:** toggles vim-style prompt editing (`Esc` → normal, `i`/`I`/`a`/`A` → insert, `h`/`l`/`0`/`^`/`$`, `x`, `Enter` sends). **Plain REPL:** prints TUI-only hint |
| Unknown | Error unless matched as a skill name |

## openclaude3 — built-in names (`COMMANDS()`)

Alphabetical **primary** `name` values from modules wired in [`commands.ts`](https://github.com/Gitlawb/openclaude/blob/main/src/commands.ts) `COMMANDS()` (exact strings users type after `/`). This set **excludes** dynamically loaded skills/plugins/workflows/MCP skills.

`add-dir`, `advisor`, `agents`, `branch`, `brief` (Kairos), `buddy` (buddy feature), `btw`, `chrome`, `clear`, `color`, `compact`, `config`, `context`, `copy`, `cost`, `desktop`, `diff`, `doctor`, `dream`, `effort`, `exit`, `export`, `extra-usage`, `fast`, `feedback`, `files`, `heapdump`, `help`, `hooks`, `ide`, `init`, `insights`, `install-github-app`, `install-slack-app`, `keybindings`, `login`, `logout`, `mcp`, `memory`, `mobile`, `model`, `onboard-github`, `output-style`, `passes`, `permissions`, `plan`, `plugin`, `pr-comments`, `privacy-settings`, `provider`, `rate-limit-options`, `reload-plugins`, `release-notes`, `remote-control` (bridge feature), `remote-env`, `rename`, `resume`, `review`, `rewind`, `sandbox`, `security-review`, `session`, `skills`, `stats`, `status`, `statusline`, `stickers`, `tag`, `tasks`, `terminal-setup`, `theme`, `think-back`, `thinkback-play`, `upgrade`, `usage`, `ultrareview`, `vim`

**Also merged into `getCommands(cwd)`** (not all are in the static list above): bundled skills, plugin skills, skill-dir commands, plugin commands, optional workflow commands, and optional dynamic skills. Some entries in `COMMANDS()` are **feature-gated** at build time — see `feature('…')` branches in `commands.ts`.

**Internal / Ant-only** built-ins live in `INTERNAL_ONLY_COMMANDS` in the same file.

## v4 non-goals (vs v3)

Login/logout, Claude.ai flows, `/review` / security-review pipelines, GitHub app install, remote bridge, voice, plugins, workflows, heapdump, and most Ink-heavy wizards are **not** targeted in v4. Use **`openclaude` subcommands**, **config**, and **MCP** instead.

## Recommended refinements (v3 → v4)

Below is a **prioritized** subset of v3 built-ins that are still missing in v4 but align with the **terminal agent + config + tools** shape. Names match v3 where it helps muscle memory; behavior can be a smaller Go implementation.

### Tier A — high value, fits v4 without a new product surface

**Status (v4):** `/config`, `/permissions`, `/version`, `/init`, and `/export` are implemented; see the v4 table above.

| v3-style command | Why | v4-shaped implementation |
|------------------|-----|---------------------------|
| **`/config`** | Users constantly ask “what is actually loaded?” | Print config **search order**, resolved file path(s), merged `provider.*` / session flags / MCP server **names** only (no secrets). Optionally `openclaude.yaml` path from [`WritableConfigPath`](../internal/config/mcp_configfile.go) when relevant. |
| **`/version`** | Matches `/exit`-style discoverability | Alias for `openclaude version` output (embed `version`/`commit` from main). |
| **`/export`** | Share or archive a conversation | Dump current in-memory transcript as **JSON** (and optionally a minimal **Markdown** turn list) to stdout or a user-given path; redact or warn on secrets. |
| **`/permissions`** | Clarifies tool policy | Print `OPENCLAUDE_AUTO_APPROVE_TOOLS`, MCP `approval` summary from config, workspace rule one-liner (see [SECURITY.md](./SECURITY.md)). |
| **`/init`** | Onboarding for new repos | Print a **starter `openclaude.yaml`** snippet + pointer to `openclaude.example.yaml` / CONFIG.md (no interactive Ink). |

### Tier B — useful, more design or plumbing

| v3-style command | Why | Notes |
|------------------|-----|--------|
| **`/files`** | Quick workspace picture without invoking the model | Thin wrapper: list cwd top-level, or `git ls-files` when `.git` exists (cap lines); same boundary rules as tools. |
| **`/stats`** | Deeper than `/cost` | Count messages by role, tool-call count from transcript, last compact time — no billing until APIs expose usage headers. |
| **`/session rename <new>`** (or **`/rename`**) | v3 users rename sessions | Add `Store.Rename` / file move under session dir + update in-memory id; watch collisions. |
| **`/release-notes` / `/upgrade`** | Discoverability | Static text: link to GitHub Releases + `go install` / binary install blurb (no auto-upgrade unless explicitly scoped later). |
| **`/diff`** | “What changed?” | Optional: `git diff --stat` or last N lines when repo is git; keep sandbox same as Bash tool. |
| **`/review` / `/security-review`** | Common workflows | **Prompt templates** only: expand to a **user message** (or print template to paste) — not a full v3 pipeline unless gRPC/CI integration is added. |

### Tier C — defer or keep out of slash router

| Area | Examples from v3 | Reason |
|------|------------------|--------|
| Auth / 1P product | `/login`, `/logout`, `/extra-usage`, rate-limit pickers | Tied to Claude.ai / billing UIs. |
| IDE / desktop | `/ide`, `/desktop`, `/chrome` | v4 is terminal-first; use MCP or external tools. |
| Plugins / dynamic UI | `/plugin`, `/reload-plugins`, `/hooks` | No v4 plugin host yet. |
| Memory / tags / tasks | `/memory`, `/tag`, `/tasks` | Needs persistent product model beyond session JSON. |
| Plan mode / effort knobs | `/plan`, `/effort`, `/fast` | Requires agreed semantics on top of OpenAI-compat parameters and agent loop. |
| Remote / voice / buddy | `/remote-control`, `/voice`, `/buddy`, … | Different binary features and feature flags. |

### Ordering suggestion for implementation

1. **`/config`** + **`/permissions`** (pure text, builds trust).  
2. **`/version`** + **`/init`** + **`/release-notes`** (small, mostly static).  
3. **`/export`** (high utility for support and migration).  
4. **`/files`**, **`/stats`**, **`/session rename`**, then prompt-template **`/review`** if desired.

When adding any of these, extend [`slash.go`](../cmd/openclaude/slash.go), [`printChatHelpTo`](../cmd/openclaude/chat.go), and this document’s v4 table.

## Maintenance

When adding or renaming a v4 slash command, update:

1. [`cmd/openclaude/slash.go`](../cmd/openclaude/slash.go) (and related `slash_*.go` if needed)
2. [`printChatHelpTo`](../cmd/openclaude/chat.go)
3. This document

v3 inventory should stay aligned with `src/commands.ts` and the `commands/*/index.ts` (or equivalent) `name` / `aliases` fields in the openclaude3 tree.
