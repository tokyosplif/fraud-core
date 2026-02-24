package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokyosplif/fraud-core/internal/config"
	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/internal/infrastructure/db"
	"github.com/tokyosplif/fraud-core/internal/infrastructure/grpc_client"
	"github.com/tokyosplif/fraud-core/internal/infrastructure/kafka"
	"github.com/tokyosplif/fraud-core/internal/usecase"
	"github.com/tokyosplif/fraud-core/pkg/closer"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	dbMaxRetries      = 30
	dbMaxOpenConns    = 25
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = time.Hour
	dbRetryDelay      = 1 * time.Second
)

func RunProcessor(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("config init: %w", err)
	}

	var pgDB *gorm.DB
	for i := 0; i < dbMaxRetries; i++ {
		pgDB, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
		if err == nil {
			break
		}
		slog.Warn("Waiting for database...", "attempt", i+1, "max", dbMaxRetries)
		time.Sleep(dbRetryDelay)
	}
	if err != nil || pgDB == nil {
		return fmt.Errorf("postgres failure after %d attempts: %w", dbMaxRetries, err)
	}

	if err := pgDB.AutoMigrate(&domain.User{}, &domain.FraudEvent{}); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err := db.SeedUsers(pgDB); err != nil {
		slog.Error("Seeding failed, but continuing execution", "err", err)
	}

	sqlDB, err := pgDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql db to configure pool: %w", err)
	}

	sqlDB.SetMaxOpenConns(dbMaxOpenConns)
	sqlDB.SetMaxIdleConns(dbMaxIdleConns)
	sqlDB.SetConnMaxLifetime(dbConnMaxLifetime)

	defer closer.Close(sqlDB, "postgres")

	slog.Info("Postgres connection pool configured",
		"max_open", dbMaxOpenConns,
		"max_idle", dbMaxIdleConns,
	)

	pgRepo := db.NewPostgresRepository(pgDB)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer closer.Close(rdb, "redis")
	redisRepo := db.NewRedisRepository(rdb)

	aiClient, err := grpc_client.NewRiskClient(cfg.RiskEngineAddr)
	if err != nil {
		return fmt.Errorf("ai client: %w", err)
	}
	defer closer.Close(aiClient, "risk.client")

	if len(cfg.KafkaBrokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}
	if err := kafka.EnsureTopics(cfg.KafkaBrokers[0], cfg.KafkaTopic, cfg.AlertsTopic); err != nil {
		return fmt.Errorf("failed to ensure kafka topics: %w", err)
	}

	publisher := kafka.NewPublisher[domain.FraudAlert](cfg.KafkaBrokers, cfg.AlertsTopic)
	defer closer.Close(publisher, "kafka.publisher")

	detector := usecase.NewFraudDetector(aiClient, pgRepo, redisRepo, publisher)

	consumer := kafka.NewConsumer[domain.Transaction](cfg.KafkaBrokers, cfg.KafkaTopic, cfg.ProcessorGroupID)
	defer closer.Close(consumer, "kafka.consumer")

	slog.Info("FRAUD CORE ENGINE started", "topic", cfg.KafkaTopic)

	return consumer.Consume(ctx, func(ctx context.Context, tx domain.Transaction) error {
		if err := detector.Detect(ctx, tx); err != nil {
			slog.Error("Failed to detect fraud", "tx_id", tx.ID, "err", err)
			return err
		}
		return nil
	})
}
