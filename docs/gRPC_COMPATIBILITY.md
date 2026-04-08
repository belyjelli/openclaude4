# OpenClaude v4 — gRPC Compatibility

This document explains the compatibility situation between OpenClaude v3 (`openclaude.v1` proto) and v4 (`openclaude.v4` proto), and provides migration guidance for clients using the gRPC API.

## Summary

**OpenClaude v4 intentionally uses a different proto package (`openclaude.v4`) than v3 (`openclaude.v1`).** This is a deliberate design decision to avoid wire-level compatibility assumptions between versions.

### Why Different Packages?

1. **Explicit versioning**: Reflection and load balancers can route by package name (`openclaude.v1.AgentService` vs `openclaude.v4.AgentService`)
2. **No hidden breaking changes**: v4 changes event names, field names, and message structures without assuming backward compatibility
3. **Clean migration path**: Clients must explicitly migrate rather than silently breaking

## Wire-Level Differences

| v3 (`openclaude.v1`) | v4 (`openclaude.v4`) | Notes |
|---------------------|---------------------|-------|
| `ChatRequest.message` | `ChatRequest.user_text` | Field renamed for clarity |
| `ActionRequired` | `PermissionRequired` | Event name changed |
| `FinalResponse` | `AssistantFinished` + `TurnComplete` | Split into two events |
| `ErrorResponse` | `ErrorEvent` | Renamed |
| `UserInput` / `CancelSignal` | Same roles | Preserved |
| `TextChunk` | Same | Preserved |
| `ToolCallStart` | Same field names | Preserved |
| `ToolCallResult` | `is_error` + `output` | Same structure; v4 adds `error_message` |
| `session_id` | Field 4 | Preserved with same semantics |
| *(v3 image parts on user message, if any)* | `ChatRequest.image_url`, `ChatRequest.image_inline` | v4 explicit fields; same OpenAI-style multimodal wire to the model as CLI `--image-url` / files |

## Migration Guide

### Step 1: Update Proto Dependencies

```bash
# Remove v3 proto dependency
go get -u github.com/gitlawb/openclaude/src/proto@v3  # if used

# Add v4 proto (internal in v4, use generated code)
# The v4 proto is in internal/grpc/proto/openclaude.proto
```

### Step 2: Update Client Code

**v3 style:**
```go
// v3 - check for ActionRequired
case *openclaude_v1.UserAction:
    handlePermissionRequired(action)
case *openclaude_v1.FinalResponse:
    handleComplete(response)
```

**v4 style:**
```go
// v4 - check for PermissionRequired
case *openclaude_v4.PermissionRequired:
    handlePermissionRequired(action)
case *openclaude_v4.AssistantFinished:
    handleAssistantFinished(response)
case *openclaude_v4.TurnComplete:
    handleTurnComplete(response)
```

### Step 3: Update Message Field Names

If you were accessing `ChatRequest.message`, update to `ChatRequest.user_text`:

```go
// v3
req := &openclaude_v1.ChatRequest{
    Message: "hello",
}

// v4
req := &openclaude_v4.ChatRequest{
    UserText: "hello",
}
```

### Step 4: Handle Split FinalResponse Events

v4 splits `FinalResponse` into two events:

1. `AssistantFinished` - emitted when the model's turn completes
2. `TurnComplete` - emitted when the entire user turn completes (after tool execution, if any)

```go
// v3: single FinalResponse event
case *openclaude_v1.FinalResponse:
    output := response.Output

// v4: two events
case *openclaude_v4.AssistantFinished:
    // Model text is complete, but tools may still run
    assistantText := response.Text

case *openclaude_v4.TurnComplete:
    // Entire turn (including tools) is complete
    turnComplete := response
```

## Compatibility Gateway (Optional)

If you need to support both v3 and v4 clients simultaneously, consider implementing a compatibility gateway:

```go
// Pseudo-code for compatibility layer
func (s *Server) ChatStream(req *openclaude_v4.ChatRequest, stream openclaude_v4.AgentService_ChatStreamServer) error {
    // Translate v4 -> v3 for legacy clients
    if req.IsLegacyV3 {
        v3Req := &openclaude_v1.ChatRequest{
            Message: req.UserText,
            // ... other translations
        }
        return s.translateToV3(v3Req, stream)
    }

    // Native v4 handling
    return s.handleV4(req, stream)
}
```

## Testing Migration

1. **Unit test**: Create a test that sends a simple message and verifies event structure
2. **Integration test**: Run a real chat with tools enabled and verify all events
3. **Edge cases**: Test interrupted turns, tool errors, and streaming behavior

## References

- [internal/grpc/README.md](../internal/grpc/README.md) - v4 gRPC API documentation
- [internal/grpc/proto/openclaude.proto](../internal/grpc/proto/openclaude.proto) - v4 proto definition
- [docs/PROTO_VERSIONING.md](./PROTO_VERSIONING.md) - Versioning strategy explanation

## See Also

- [MIGRATION_V3.md](./MIGRATION_V3.md) - General v3 to v4 migration guide
- [CONFIG.md](./CONFIG.md) - Configuration changes between versions
