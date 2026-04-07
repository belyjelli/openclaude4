---
name: transcript-secret-redaction
description: Implements secret redaction for OpenClaude v4 transcripts and logged tool/model content. Use for TODO "Secret scanning / redaction in transcripts" or when avoiding API keys in output, logs, or future session files.
---

You implement **secret redaction** so sensitive material does not appear verbatim in transcripts or structured logs the product persists or echoes.

When invoked:

1. Find where **messages**, **tool arguments/results**, or **stream chunks** are assembled for display or persistence (`internal/core`, `cmd/openclaude`, relevant `internal/tools`). Prefer a **single redaction boundary** (or a small shared helper package) rather than scattering regexes.
2. Redact common patterns: bearer tokens, `Authorization` headers, `api_key`-like fields, long base64-ish secrets—balance false positives vs safety; document limitations in **docs/SECURITY.md** briefly.
3. **In scope:** `internal/core/`, `internal/tools/`, `cmd/openclaude/` only as needed for the pipeline; tests in the same packages.
4. **Out of scope:** Full TUI (`internal/tui/`), gRPC server, npm workspace. Do not own **session file format** unless the user asked both—if sessions land later, expose a reusable `redactForDisplay`/`redactForStorage` API.
5. **Done when:** `go test ./...` passes; new behavior covered by **unit tests** (table-driven); update **TODO.md** checkbox only when the feature is complete.

Follow project style: minimal diff, no drive-by refactors, errcheck and lint clean.
