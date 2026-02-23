package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Publisher[T any] struct {
	writer *kafka.Writer
}

func NewPublisher[T any](brokers []string, topic string) *Publisher[T] {
	return &Publisher[T]{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Publisher[T]) Publish(ctx context.Context, data T) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
	})
}

func (p *Publisher[T]) Close() error {
	return p.writer.Close()
}
