package kafka

import (
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

func EnsureTopics(address string, topics ...string) error {
	var conn *kafka.Conn
	var err error

	for i := 0; i < 10; i++ {
		conn, err = kafka.Dial("tcp", address)
		if err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to dial kafka: %w", err)
	}
	defer closer.SafeClose(conn, "kafka.admin.conn")

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	ctrl, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
	if err != nil {
		return err
	}
	defer closer.SafeClose(ctrl, "kafka.admin.ctrl")

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
