package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers     []string
	KafkaTopic       string
	AlertsTopic      string
	RedisAddr        string
	PostgresDSN      string
	RiskEngineAddr   string
	DashboardPort    string
	DashboardGroupID string
}

func New() (*Config, error) {
	cfg := &Config{
		KafkaBrokers:     strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		KafkaTopic:       getEnv("KAFKA_TOPIC", "raw-transactions"),
		AlertsTopic:      getEnv("ALERTS_TOPIC", "fraud-alerts"),
		RedisAddr:        getEnv("REDIS_ADDR", "redis:6379"),
		PostgresDSN:      getEnv("POSTGRES_DSN", "host=postgres port=5432 user=user password=password dbname=fraud_radar sslmode=disable"),
		RiskEngineAddr:   getEnv("RISK_ENGINE_ADDR", "ai-risk-engine:50051"),
		DashboardPort:    getEnv("DASHBOARD_PORT", ":8080"),
		DashboardGroupID: getEnv("DASHBOARD_GROUP_ID", "dashboard-group"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if len(c.KafkaBrokers) == 0 || c.KafkaBrokers[0] == "" {
		return fmt.Errorf("CRITICAL: KAFKA_BROKERS is required")
	}
	if c.PostgresDSN == "" {
		return fmt.Errorf("CRITICAL: POSTGRES_DSN is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
