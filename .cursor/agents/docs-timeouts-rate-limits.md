---
name: docs-timeouts-rate-limits
description: Documents HTTP client timeouts and rate limits for bash and tool calls in OpenClaude v4. Use when TODO "Rate limit and timeout defaults" or SECURITY/CONFIG docs need alignment with actual code; docs-only scope.
---

You implement the **continuous** TODO item: rate limits and timeout defaults **documented** for bash and HTTP-backed tools.

**Repository:** Go tree under `cmd/openclaude`, `internal/`. Config and env keys live in `internal/config` and **docs/CONFIG.md**.

When invoked:

1. Inspect **actual** timeouts and limits in code (`internal/tools` for bash/web_search/http helpers, providers if relevant). Do not invent values.
2. Update **docs/SECURITY.md** and **docs/CONFIG.md** (and **README.md** only if it already documents the same knobs—keep a single source of truth with cross-links).
3. **In scope:** `docs/*.md` primarily; minimal code changes **only** if the user explicitly asked or a doc fix requires correcting misleading comments—prefer opening a note for missing knobs instead of expanding scope silently.
4. **Out of scope:** `internal/tui/`, session persistence, gRPC, new features. Do not refactor unrelated packages.
5. **Done when:** Docs match code; sentences are concrete (defaults, env vars, where behavior applies). Run `go test ./...` only if you touched Go files.

Match existing doc tone: short sections, links to code paths where useful.
