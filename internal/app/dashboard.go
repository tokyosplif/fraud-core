package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/internal/config"
	transport "github.com/tokyosplif/fraud-core/internal/delivery/http"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

func RunDashboard(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.AlertsTopic,
		GroupID: cfg.DashboardGroupID,
	})
	defer closer.SafeClose(reader, "kafka.reader")

	mux := http.NewServeMux()

	transport.RegisterRoutes(mux, reader)

	server := &http.Server{
		Addr:    cfg.DashboardPort,
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Dashboard server starting", "addr", cfg.DashboardPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		slog.Info("Shutting down dashboard server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
