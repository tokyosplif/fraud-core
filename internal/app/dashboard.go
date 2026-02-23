package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tokyosplif/fraud-core/internal/config"
	transport "github.com/tokyosplif/fraud-core/internal/delivery/http"
	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/internal/infrastructure/kafka"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

func RunDashboard(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.KafkaBrokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}
	if err := kafka.EnsureTopics(cfg.KafkaBrokers[0], cfg.KafkaTopic, cfg.AlertsTopic); err != nil {
		return fmt.Errorf("failed to ensure kafka topics: %w", err)
	}

	hub := transport.NewHub()
	consumer := kafka.NewConsumer[domain.FraudAlert](cfg.KafkaBrokers, cfg.AlertsTopic, cfg.DashboardGroupID)
	defer closer.Close(consumer, "kafka.consumer")

	mux := http.NewServeMux()
	transport.RegisterRoutes(mux, hub)

	server := &http.Server{
		Addr:    cfg.DashboardPort,
		Handler: mux,
	}

	go func() {
		slog.Info("Kafka Consumer for Dashboard started", "topic", cfg.AlertsTopic)
		err := consumer.Consume(ctx, func(ctx context.Context, alert domain.FraudAlert) error {
			hub.BroadcastAlert(alert)
			return nil
		})
		if err != nil {
			slog.Error("Kafka consumer fatal error", "err", err)
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Dashboard UI available at", "port", cfg.DashboardPort)
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
