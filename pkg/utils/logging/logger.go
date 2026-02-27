package logging

import (
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/goerr/v2"
)

var defaultLogger *slog.Logger

func init() {
	defaultLogger = New(os.Stderr, slog.LevelInfo)
}

// New creates a new slog.Logger with clog handler for colored console output.
func New(w io.Writer, level slog.Level) *slog.Logger {
	handler := clog.New(
		clog.WithWriter(w),
		clog.WithLevel(level),
	)
	return slog.New(handler)
}

// Default returns the default logger.
func Default() *slog.Logger {
	return defaultLogger
}

// SetDefault sets the default logger.
func SetDefault(l *slog.Logger) {
	defaultLogger = l
}

// ErrAttr creates a slog.Attr for logging errors with goerr values expansion.
func ErrAttr(err error) slog.Attr {
	var goErr *goerr.Error
	if errors.As(err, &goErr) {
		attrs := []any{slog.String("message", goErr.Error())}
		for k, v := range goErr.Values() {
			attrs = append(attrs, slog.Any(k, v))
		}
		return slog.Group("error", attrs...)
	}
	return slog.Any("error", err)
}
