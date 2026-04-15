package tools

import "context"

type workdirKey struct{}

type toolCallIDKey struct{}

type subTaskDepthKey struct{}

// WithWorkDir returns a child context carrying the absolute workspace root.
// Tools resolve relative paths against this directory and refuse paths that escape it.
func WithWorkDir(ctx context.Context, absRoot string) context.Context {
	return context.WithValue(ctx, workdirKey{}, absRoot)
}

// WorkDir returns the workspace root from ctx, or empty if unset.
func WorkDir(ctx context.Context) string {
	v, _ := ctx.Value(workdirKey{}).(string)
	return v
}

// WithSubTaskDepth records nested Task depth in context (policy hooks; nested Task runs omit the Task tool).
// TUI transcript nesting uses core.Agent.EventSubTaskDepth / core.Event.SubTaskDepth, not this context value.
func WithSubTaskDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, subTaskDepthKey{}, depth)
}

// SubTaskDepth returns the current Task nesting depth (0 outside any Task).
func SubTaskDepth(ctx context.Context) int {
	v, _ := ctx.Value(subTaskDepthKey{}).(int)
	return v
}

// WithToolCallID records the OpenAI tool_call id for the current tool execution (streaming / audit).
func WithToolCallID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, toolCallIDKey{}, id)
}

// ToolCallID returns the tool call id from [WithToolCallID], or empty.
func ToolCallID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(toolCallIDKey{}).(string)
	return v
}
