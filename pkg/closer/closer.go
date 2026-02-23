package closer

import (
	"io"
	"log/slog"
)

func Close(c io.Closer, name string) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		slog.Error("failed to close resource",
			"resource", name,
			"error", err,
		)
	}
}
