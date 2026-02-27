package logging

import (
	"context"
	"log/slog"
)

type contextKey struct{}

// With stores the logger in the context.
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// From retrieves the logger from the context. Returns the default logger if not set.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return l
	}
	return Default()
}
