package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokyosplif/fraud-core/internal/app"
	"github.com/tokyosplif/fraud-core/pkg/logger"
)

func main() {
	logger.Setup()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunDashboard(ctx); err != nil {
		slog.Error("Dashboard stopped with error", "err", err)
		os.Exit(1)
	}
}
