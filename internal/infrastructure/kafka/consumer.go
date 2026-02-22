package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/internal/domain"
	_ "github.com/tokyosplif/fraud-core/pkg/closer"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
	}
}

func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, domain.Transaction) error) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			slog.Error("Kafka read error", "err", err)
			return err
		}

		var tx domain.Transaction
		if err := json.Unmarshal(m.Value, &tx); err != nil {
			slog.Error("JSON parse error", "err", err)
			continue
		}

		if err := handler(ctx, tx); err != nil {
			slog.Error("Handler process failed", "tx_id", tx.ID, "err", err)
		}
	}
}
