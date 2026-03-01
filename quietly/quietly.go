// Package quietly providers helpers so we can enable errcheck without too much pain for commonly ignored errors.
package quietly

import (
	"io"
	"log/slog"
)

func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		// we can't do anything about this error
		slog.Info("failed to close resource", "error", err)
	}
}
