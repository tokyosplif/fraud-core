package app

import (
	"context"
	"time"

	"github.com/tokyosplif/fraud-core/internal/config"
	"github.com/tokyosplif/fraud-core/internal/infrastructure/kafka"
	"github.com/tokyosplif/fraud-core/internal/usecase"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

func RunSimulator(ctx context.Context) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer closer.SafeClose(kafkaProducer, "kafka.producer")

	simulator := usecase.NewSimulator(kafkaProducer)

	simulator.Run(ctx)

	return nil
}
