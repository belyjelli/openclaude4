package tools

import "context"

type workdirKey struct{}

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
