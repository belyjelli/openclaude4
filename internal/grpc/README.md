# OpenClaude v4 — gRPC API (Phase 6)

Headless API that drives the same [`internal/core.Agent`](../../internal/core/agent.go) loop as the CLI and TUI: one [`ChatRequest`](proto/openclaude.proto) per user turn, server events aligned with [`core.Event`](../../internal/core/event.go) kinds.

## Layout

| Path | Purpose |
|------|---------|
| [`proto/openclaude.proto`](proto/openclaude.proto) | `openclaude.v4` service + messages |
| [`openclaudev4/`](openclaudev4/) | Generated `protoc-gen-go` + `protoc-gen-go-grpc` output (**do not hand-edit**) |
| [`server.go`](server.go) | [`Register`](server.go) + [`AgentService`](server.go) implementation |
| [`server_test.go`](server_test.go) | In-process test with [`bufconn`](https://pkg.go.dev/google.golang.org/grpc/test/bufconn) + `httptest` model |

Regenerate stubs after changing the proto (from repo root, with `protoc-gen-go` and `protoc-gen-go-grpc` on `PATH`):

```bash
protoc -I internal/grpc/proto \
  --go_out=internal/grpc/openclaudev4 --go_opt=paths=source_relative \
  --go-grpc_out=internal/grpc/openclaudev4 --go-grpc_opt=paths=source_relative \
  internal/grpc/proto/openclaude.proto
```

## Versioning vs OpenClaude v3 (`openclaude.proto`)

The TypeScript v3 service uses **`package openclaude.v1`** (historical file: `src/proto/openclaude.proto` in the v3 codebase).

v4 intentionally uses **`package openclaude.v4`** and a distinct Go import path under `internal/grpc/openclaudev4` so:

- v3 and v4 clients do not assume wire compatibility.
- The service **name** differs (`openclaude.v4.AgentService` vs `openclaude.v1.AgentService`), so reflection and load balancers can route by package.

**Conceptual mapping** (not byte-for-byte identical):

| v3 (`openclaude.v1`) | v4 (`openclaude.v4`) |
|----------------------|----------------------|
| `ChatRequest.message` | `ChatRequest.user_text` |
| `ChatRequest.working_directory` | same |
| `ChatRequest.model` | same (optional; server process still owns the real client today) |
| `ChatRequest.session_id` | `ChatRequest.session_id` (field 4) — on-disk binding when `openclaude serve` runs with sessions enabled (same dir as REPL) |
| `UserInput` / `CancelSignal` | same roles |
| `TextChunk` | same |
| `ToolCallStart` | same field names (`tool_use_id`, …) |
| `ToolCallResult` | v4 adds `error_message`; `is_error` + `output` aligned |
| `ActionRequired` | `PermissionRequired` (+ explicit `PermissionAck` after kernel `KindPermissionResult`) |
| `FinalResponse` | split: `AssistantFinished` per model round + `TurnComplete` when the user turn ends |
| `ErrorResponse` | `ErrorEvent` |

## CLI: `openclaude serve`

Implemented in [`cmd/openclaude/serve.go`](../../cmd/openclaude/serve.go): same bootstrap as chat (config, `NewStreamClient`, default registry, MCP, `Task` tool with an atomic parent slot), then [`ocrpc.Register`](server.go) and TCP listen.

- **Address:** `--listen` or env **`OPENCLAUDE_GRPC_ADDR`** (default **`:50051`**).
- **Sessions:** [`Kernel.Session`](server.go) uses `config.SessionDisabled()` and `config.EffectiveSessionDir()`. [`ChatRequest.session_id`](proto/openclaude.proto) selects the on-disk session; empty id on a stream uses a new random id for the first turn, then sticks for later empty requests on that stream.
- **Concurrency:** one `RunUserTurn` at a time server-wide (`serveTurnMu`) so the `Task` tool always resolves the active agent.
- **Future:** TLS / auth middleware; optional HTTP gateway.

## Tests

```bash
go test ./internal/grpc/...
```
