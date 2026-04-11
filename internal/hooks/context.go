package hooks

import "context"

type ctxKey int

const sessionIDKey ctxKey = 1

// WithSessionID attaches a session identifier for hook dispatch (REPL/TUI transcript id).
func WithSessionID(parent context.Context, id string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, sessionIDKey, id)
}

// SessionIDFrom returns the session id previously attached with [WithSessionID], or "".
func SessionIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	s, _ := ctx.Value(sessionIDKey).(string)
	return s
}
