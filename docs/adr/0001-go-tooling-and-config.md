# ADR 0001 — Go version, releases, and configuration compatibility

## Status

Accepted (April 2026)

## Context

OpenClaude v4 is a Go rewrite of the v3 TypeScript/Bun CLI. Contributors need a single place to learn: which Go version to target, how binaries are produced, and how v4 relates to v3 configuration files.

## Decision

1. **Go version** — The module `go` directive in `go.mod` is authoritative (currently 1.22+ minimum alignment with CI `go-version-file`; patch upgrades follow CI and `golangci-lint` support).

2. **Releases** — Primary distribution is **static-ish single binaries** via [Goreleaser](https://goreleaser.com/) (`goreleaser.yml`). Semver tags drive release artifacts; detailed release checklist remains in project TODO / future `RELEASE.md`.

3. **v3 configuration** — v4 **may merge** `~/.openclaude-profile.json` and `./.openclaude-profile.json` (see `internal/config/profile_v3.go`) into the same viper layer as YAML, with **lower precedence** than `openclaude.yaml` / `--config` and **lower than** environment variables and CLI flags. v3 `settings.json` is **not** read automatically; map important fields manually or extend v4 loaders later.

4. **Precedence** — Documented in `internal/config/load.go` and [docs/CONFIG.md](../CONFIG.md): flags → env → config file → v3 profile → implicit defaults in getters.

## Consequences

- CI and local dev must use a Go toolchain compatible with `go.mod`.
- Users migrating from v3 should keep `.openclaude-profile.json` for a smoother path but prefer `openclaude.yaml` + env for secrets.
- Unknown `provider.name` values fail fast via `config.Validate()` before chat starts.
