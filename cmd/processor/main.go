package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokyosplif/fraud-core/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunProcessor(ctx); err != nil {
		log.Fatalf("Processor stopped with error: %v", err)
	}
}
