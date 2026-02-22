package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/internal/domain"
)

type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, alert domain.FraudAlert) error {
	bytes, err := json.Marshal(alert)
	if err != nil {
		log.Printf("failed to marshal fraud alert: %v", err)
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(alert.TransactionID),
		Value: bytes,
	})
	if err != nil {
		log.Printf("failed to publish alert to kafka: %v", err)
		return err
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
