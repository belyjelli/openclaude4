# Contributing to OpenClaude v4

## Prerequisites

- Go **1.22+** (see `go.mod` for the version used in CI).

## Build and run

```bash
go build -ldflags "-X main.version=0.0.0-dev -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o openclaude ./cmd/openclaude
./openclaude
```

Chat mode expects `OPENAI_API_KEY`. Optional: `OPENAI_BASE_URL`, `OPENAI_MODEL`, or flags `--model` / `--base-url`.

## Tests and checks

```bash
go test ./...
go vet ./...
```

Lint (same as CI):

```bash
bash -c 'curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.9.0'
"$(go env GOPATH)/bin/golangci-lint" run
```

Or use a locally installed `golangci-lint` matching `.golangci.yml`.

## Pull requests

- Keep changes focused on a single concern when possible.
- Run `go test ./...` and ensure CI would pass (`go vet`, `golangci-lint`).
- For agent or tool behavior, add or extend tests under `internal/core`, `internal/tools`, or relevant packages.

## Architecture pointers

- [`docs/DESIGN.md`](./docs/DESIGN.md) — module boundaries and goals.
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — phased delivery.
- [`TODO.md`](./TODO.md) — checklist aligned with the Go codebase.
