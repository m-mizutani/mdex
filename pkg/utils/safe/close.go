package safe

import (
	"context"
	"io"

	"github.com/m-mizutani/mdex/pkg/utils/logging"
)

// Close closes the given io.Closer and logs a warning if an error occurs.
func Close(ctx context.Context, c io.Closer) {
	if err := c.Close(); err != nil {
		logging.From(ctx).Warn("failed to close resource", "error", err)
	}
}
