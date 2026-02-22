package app

import (
	"context"
	"log"
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
		return err
	}

	var pgDB *gorm.DB
	for i := 0; i < 60; i++ {
		pgDB, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("[WAIT] Postgres not ready, retrying... (%d/60) err=%v", i+1, err)
		time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
	}
	if err != nil {
		return err
	}

	if err := pgDB.AutoMigrate(&domain.User{}, &domain.FraudEvent{}); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
	pgRepo := db.NewPostgresRepository(pgDB)
	_ = db.SeedUsers(pgDB)

	for i := 0; i < 20; i++ {
		err = kafka.EnsureTopics(cfg.KafkaBrokers[0], cfg.KafkaTopic, cfg.AlertsTopic)
		if err == nil {
			break
		}
		log.Printf("[WAIT] Kafka not ready, retrying... (%d/20)", i+1)
		time.Sleep(3 * time.Second)
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer closer.SafeClose(rdb, "redis.client")
	rdRepo := db.NewRedisRepository(rdb)

	aiClient, err := grpc_client.NewRiskClient(cfg.RiskEngineAddr)
	if err != nil {
		log.Fatalf("failed to connect to risk engine: %v", err)
	}
	defer closer.SafeClose(aiClient, "risk.client")

	publisher := kafka.NewPublisher(cfg.KafkaBrokers, cfg.AlertsTopic)
	detector := usecase.NewFraudDetector(aiClient, pgRepo, rdRepo, publisher)
	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, "fraud-group")

	log.Println("🚀 FRAUD CORE ENGINE is running and healthy!")
	return consumer.Consume(ctx, detector.Detect)
}
