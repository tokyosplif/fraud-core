package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/internal/domain"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) Send(ctx context.Context, tx domain.Transaction) error {
	bytes, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal tx: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(tx.UserID),
		Value: bytes,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
