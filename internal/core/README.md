# internal/core

Agent kernel: multi-turn tool loop, OpenAI-style streaming, and a typed **event harness** for transports.

## Events (`event.go`)

Set [`Agent.OnEvent`](./agent.go) to receive [`Event`](./event.go) values with [`EventKind`](./event.go) (`user_message`, `assistant_text_delta`, `assistant_finished`, `tool_call`, `permission_*`, `tool_result`, `error`, `turn_complete`, …). The stdin REPL leaves `OnEvent` nil and only writes assistant text to [`Agent.Out`](./agent.go).

Bubble Tea / gRPC bridges should subscribe here instead of parsing stdout.

## Tests

[`agent_test.go`](./agent_test.go) — httptest SSE, tools, iterations.  
[`event_harness_test.go`](./event_harness_test.go) — event ordering for text-only, tool rounds, and stream errors.
