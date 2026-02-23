package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Consumer[T any] struct {
	reader *kafka.Reader
}

func NewConsumer[T any](brokers []string, topic, groupID string) *Consumer[T] {
	return &Consumer[T]{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
	}
}

func (c *Consumer[T]) Consume(ctx context.Context, handler func(context.Context, T) error) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				slog.Error("Kafka read error", "err", err)
				return err
			}
		}

		var data T
		if err := json.Unmarshal(m.Value, &data); err != nil {
			slog.Error("JSON parse error", "err", err, "topic", m.Topic)
			continue
		}

		if err := handler(ctx, data); err != nil {
			slog.Error("Handler process failed", "err", err)
		}
	}
}

func (c *Consumer[T]) Close() error {
	return c.reader.Close()
}
