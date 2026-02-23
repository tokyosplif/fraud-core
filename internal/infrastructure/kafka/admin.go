package kafka

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

const (
	kafkaMaxRetries = 10
	kafkaRetryDelay = 3 * time.Second
)

func EnsureTopics(address string, topics ...string) error {
	var conn *kafka.Conn
	var err error

	for i := 0; i < kafkaMaxRetries; i++ {
		conn, err = kafka.Dial("tcp", address)
		if err == nil {
			break
		}
		slog.Warn("Waiting for Kafka to be ready...", "attempt", i+1, "max", kafkaMaxRetries)
		time.Sleep(kafkaRetryDelay)
	}
	if err != nil {
		return fmt.Errorf("failed to dial kafka after %d attempts: %w", kafkaMaxRetries, err)
	}
	defer closer.Close(conn, "kafka.admin.conn")

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	ctrl, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
	if err != nil {
		return err
	}
	defer closer.Close(ctrl, "kafka.admin.ctrl")

	existing, err := ctrl.ReadPartitions()
	if err != nil {
		return err
	}

	existingTopics := make(map[string]struct{})
	for _, p := range existing {
		existingTopics[p.Topic] = struct{}{}
	}

	var toCreate []kafka.TopicConfig
	for _, t := range topics {
		if _, ok := existingTopics[t]; !ok {
			toCreate = append(toCreate, kafka.TopicConfig{Topic: t, NumPartitions: 1, ReplicationFactor: 1})
		}
	}

	if len(toCreate) > 0 {
		if err := ctrl.CreateTopics(toCreate...); err != nil {
			return err
		}
	}

	return nil
}
