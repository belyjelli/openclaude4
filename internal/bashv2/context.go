package bashv2

import "context"

type sessionCtxKey struct{}

// WithSession attaches a Bash v2 session to ctx (workspace-scoped snapshot + policy).
func WithSession(ctx context.Context, s *Session) context.Context {
	if s == nil {
		return ctx
	}
	return context.WithValue(ctx, sessionCtxKey{}, s)
}

// FromContext returns the session attached with [WithSession], or nil.
func FromContext(ctx context.Context) *Session {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(sessionCtxKey{}).(*Session)
	return v
}

type streamHookKey struct{}

// StreamHook emits incremental Bash stdout/stderr for kernel/TUI consumers.
type StreamHook func(toolCallID, chunk string, totalBytes int)

// WithStreamHook attaches a per-turn streaming callback (tool execution only).
func WithStreamHook(ctx context.Context, fn StreamHook) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, streamHookKey{}, fn)
}

// StreamHookFromContext returns the hook from [WithStreamHook], or nil.
func StreamHookFromContext(ctx context.Context) StreamHook {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(streamHookKey{}).(StreamHook)
	return v
}
