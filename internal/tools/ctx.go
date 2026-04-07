package tools

import "context"

type workdirKey struct{}

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

// WithSubTaskDepth records nested Task (sub-agent) depth for [core.TaskTool].
func WithSubTaskDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, subTaskDepthKey{}, depth)
}

// SubTaskDepth returns the current Task nesting depth (0 outside any Task).
func SubTaskDepth(ctx context.Context) int {
	v, _ := ctx.Value(subTaskDepthKey{}).(int)
	return v
}
