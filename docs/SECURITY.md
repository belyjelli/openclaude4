# Security notes (OpenClaude v4)

This document summarizes how the Go CLI limits filesystem and shell access today. It is **not** a full threat model.

## Workspace boundary

File tools (`FileRead`, `FileWrite`, `FileEdit`) and directory-scoped tools (`Grep` search path, `Glob` root) resolve paths against a **workspace root**: the process working directory at startup, unless overridden via context (tests use an explicit root).

- Relative paths are interpreted under that root.
- Absolute paths are allowed only if they still lie **inside** the same root after `filepath.Clean`; otherwise the call fails with an error that mentions escaping the workspace.
- Logic lives in `internal/tools/paths.go` (`resolveUnderWorkdir`). Regression tests are in `internal/tools/workspace_boundary_test.go`.

**Symlinks:** resolution does **not** use `filepath.EvalSymlinks`. A symlink *inside* the workspace can still point **outside** it; the kernel does not walk that link when checking the prefix, so reads/writes may follow the link to an unintended location. Treat untrusted symlinks in the workspace as out of scope for the simple prefix check, or extend the resolver with evaluated paths if you need stronger guarantees.

## Dangerous tools

`Bash`, `FileWrite`, and `FileEdit` are marked dangerous. The stdin REPL prompts for confirmation unless `OPENCLAUDE_AUTO_APPROVE_TOOLS` is set to `1` or `true` (development convenience only).

Shell commands run with a timeout and workspace-oriented working directory; they are still **full shell** invocations—users should not auto-approve in untrusted environments.

## Network

`WebSearch` and LLM providers perform outbound HTTP(S). Use API keys and base URLs you trust; prefer documented timeouts where the code sets them (e.g. HTTP client timeouts in tools).

## Ongoing work

See [TODO.md](../TODO.md): secret redaction in transcripts, dependency update policy, and other quality items.
