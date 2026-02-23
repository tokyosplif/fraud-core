package closer

import (
	"io"
	"log/slog"
)

func SafeClose(c io.Closer, name string) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		slog.Warn("Failed to close resource", "resource", name, "err", err)
	} else {
		slog.Debug("Resource closed successfully", "resource", name)
	}
}
