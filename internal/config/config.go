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
	RedisPassword    string
	PostgresDSN      string
	RiskEngineAddr   string
	DashboardPort    string
	DashboardGroupID string
	ProcessorGroupID string
}

func New() (*Config, error) {
	cfg := &Config{
		KafkaBrokers:     strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ","),
		KafkaTopic:       getEnv("KAFKA_TOPIC", "raw-transactions"),
		AlertsTopic:      getEnv("ALERTS_TOPIC", "fraud-alerts"),
		RedisAddr:        getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		PostgresDSN:      getEnv("POSTGRES_DSN", "host=postgres port=5432 user=user password=password dbname=fraud_radar sslmode=disable"),
		RiskEngineAddr:   getEnv("RISK_ENGINE_ADDR", "ai-risk-engine:50051"),
		DashboardPort:    getEnv("DASHBOARD_PORT", ":8080"),
		DashboardGroupID: getEnv("DASHBOARD_GROUP_ID", "dashboard-group"),
		ProcessorGroupID: getEnv("PROCESSOR_GROUP_ID", "fraud-processor-v3"),
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
	if c.RedisPassword == "" {
		fmt.Println("WARNING: REDIS_PASSWORD is not set")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
