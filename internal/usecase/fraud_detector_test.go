package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/tokyosplif/fraud-core/internal/domain"
)

type mockRepo struct{}

func (m *mockRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return &domain.User{ID: "user-1", RiskScore: 10}, nil
}

func (m *mockRepo) CreateUser(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *mockRepo) SaveFraudEvent(ctx context.Context, event *domain.FraudEvent) error {
	return nil
}

func (m *mockRepo) GetUserStats(ctx context.Context, userID string) (float64, float64, error) {
	return 1000, 200, nil
}

type mockCache struct {
	velocity int
}

func (m *mockCache) GetVelocity(ctx context.Context, userID string) (int, error) {
	return m.velocity, nil
}

func (m *mockCache) IncrementVelocity(ctx context.Context, userID, location string) error {
	return nil
}

func (m *mockCache) GetUserStats(ctx context.Context, userID string) (float64, float64, error) {
	return 0, 0, nil
}

func (m *mockCache) SetUserStats(ctx context.Context, userID string, maxAmount, avgAmount float64) error {
	return nil
}

func (m *mockCache) GetRiskCache(ctx context.Context, userID, merchant string) (*domain.FraudAlert, error) {
	return nil, nil
}

func (m *mockCache) SetRiskCache(ctx context.Context, userID, merchant string, alert domain.FraudAlert) error {
	return nil
}

type mockAI struct{}

func (m *mockAI) Analyze(ctx context.Context, tx domain.Transaction, user domain.User) (domain.FraudAlert, error) {
	return domain.FraudAlert{
		IsBlocked: false,
		Reason:    "Looks like a normal transaction",
	}, nil
}

type mockPublisher struct {
	PublishedAlert domain.FraudAlert
}

func (m *mockPublisher) Publish(ctx context.Context, alert domain.FraudAlert) error {
	m.PublishedAlert = alert
	return nil
}

func TestFraudDetector_VelocityBlockOverridesAI(t *testing.T) {
	repo := &mockRepo{}
	cache := &mockCache{velocity: 15}
	aiClient := &mockAI{}
	publisher := &mockPublisher{}

	detector := NewFraudDetector(aiClient, repo, cache, publisher)

	tx := domain.Transaction{
		ID:       "tx-999",
		UserID:   "user-1",
		Amount:   150.0,
		Merchant: "Test Store",
		Location: "Kyiv",
	}

	err := detector.Detect(context.Background(), tx)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	result := publisher.PublishedAlert

	if !result.IsBlocked {
		t.Errorf("Expected transaction to be blocked due to velocity, but it was allowed")
	}

	if !strings.Contains(result.Reason, "[Velocity Block]") {
		t.Errorf("Expected reason to contain '[Velocity Block]', got: %s", result.Reason)
	}
}

func TestFraudDetector_NormalTransactionIsAllowed(t *testing.T) {
	repo := &mockRepo{}
	cache := &mockCache{velocity: 2}
	aiClient := &mockAI{}
	publisher := &mockPublisher{}

	detector := NewFraudDetector(aiClient, repo, cache, publisher)

	tx := domain.Transaction{
		ID:     "tx-888",
		UserID: "user-2",
		Amount: 50.0,
	}

	err := detector.Detect(context.Background(), tx)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	result := publisher.PublishedAlert

	if result.IsBlocked {
		t.Errorf("Expected normal transaction to be allowed, but it was blocked")
	}
}
