package dryrun

import "context"

type contextKey struct{}

// WithDryRun returns a new context with dry-run mode enabled.
func WithDryRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, true)
}

// IsDryRun returns whether dry-run mode is enabled in the context.
func IsDryRun(ctx context.Context) bool {
	v, ok := ctx.Value(contextKey{}).(bool)
	return ok && v
}
