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

	if err := app.RunSimulator(ctx); err != nil {
		slog.Error("Simulator fatal error", "err", err)
		os.Exit(1)
	}

	slog.Info("Simulator stopped gracefully")
}
