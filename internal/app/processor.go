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

func RunProcessor(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("config init: %w", err)
	}

	var pgDB *gorm.DB
	for i := 0; i < 30; i++ {
		pgDB, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil || pgDB == nil {
		return fmt.Errorf("postgres failure: %w", err)
	}

	if err := pgDB.AutoMigrate(&domain.User{}, &domain.FraudEvent{}); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	pgRepo := db.NewPostgresRepository(pgDB)
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer closer.SafeClose(rdb, "redis")

	aiClient, err := grpc_client.NewRiskClient(cfg.RiskEngineAddr, rdb)
	if err != nil {
		return fmt.Errorf("ai client: %w", err)
	}
	defer closer.SafeClose(aiClient, "risk.client")

	if len(cfg.KafkaBrokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}
	if err := kafka.EnsureTopics(cfg.KafkaBrokers[0], cfg.KafkaTopic, cfg.AlertsTopic); err != nil {
		return fmt.Errorf("failed to ensure kafka topics: %w", err)
	}

	publisher := kafka.NewPublisher[domain.FraudAlert](cfg.KafkaBrokers, cfg.AlertsTopic)
	defer closer.SafeClose(publisher, "kafka.publisher")

	detector := usecase.NewFraudDetector(aiClient, pgRepo, db.NewRedisRepository(rdb), publisher)

	consumer := kafka.NewConsumer[domain.Transaction](cfg.KafkaBrokers, cfg.KafkaTopic, cfg.ProcessorGroupID)
	defer closer.SafeClose(consumer, "kafka.consumer")

	slog.Info("FRAUD CORE ENGINE started", "topic", cfg.KafkaTopic)

	return consumer.Consume(ctx, func(ctx context.Context, tx domain.Transaction) error {
		user, _ := pgRepo.GetUserByID(ctx, tx.UserID)
		if user == nil {
			user = &domain.User{ID: tx.UserID, RiskScore: 15}
			_ = pgRepo.CreateUser(ctx, user)
		}

		statsKey := fmt.Sprintf("user_stats:%s", tx.UserID)
		var stats struct {
			MaxAmount float64 `gorm:"column:max_amount"`
			AvgAmount float64 `gorm:"column:avg_amount"`
		}

		cached, err := rdb.Get(ctx, statsKey).Result()
		if err != nil || cached == "" {
			pgDB.Raw(`
					SELECT COALESCE(MAX(amount),0) as max_amount, COALESCE(AVG(amount),0) as avg_amount 
					FROM fraud_events WHERE user_id = ?`, tx.UserID).Scan(&stats)

			rdb.Set(ctx, statsKey, fmt.Sprintf("%.2f|%.2f", stats.MaxAmount, stats.AvgAmount), 10*time.Minute)
		} else {
			_, _ = fmt.Sscanf(cached, "%f|%f", &stats.MaxAmount, &stats.AvgAmount)
		}

		user.MaxTx = stats.MaxAmount
		user.AvgTx = stats.AvgAmount

		if err := detector.Detect(ctx, tx, *user); err != nil {
			slog.Error("Failed to detect fraud", "tx_id", tx.ID, "err", err)
			return err
		}

		return nil
	})
}
