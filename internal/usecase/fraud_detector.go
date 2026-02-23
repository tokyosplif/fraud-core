package usecase

import (
	"context"
	"log/slog"

	"github.com/tokyosplif/fraud-core/internal/domain"
)

type AIClient interface {
	Analyze(ctx context.Context, tx domain.Transaction, user domain.User) (domain.FraudAlert, error)
}

type Repository interface {
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	SaveFraudEvent(ctx context.Context, event *domain.FraudEvent) error
}

type RateLimitRepository interface {
	GetVelocity(ctx context.Context, userID string) (int, error)
	IncrementVelocity(ctx context.Context, userID, location string) error
}

type FraudPublisher interface {
	Publish(ctx context.Context, alert domain.FraudAlert) error
}

type FraudDetector struct {
	aiClient  AIClient
	repo      Repository
	cache     RateLimitRepository
	publisher FraudPublisher
}

func NewFraudDetector(ai AIClient, r Repository, c RateLimitRepository, p FraudPublisher) *FraudDetector {
	return &FraudDetector{
		aiClient:  ai,
		repo:      r,
		cache:     c,
		publisher: p,
	}
}

func (d *FraudDetector) Detect(ctx context.Context, tx domain.Transaction, user domain.User) error {
	isVelocityFraud := false
	if d.cache != nil {
		vel, _ := d.cache.GetVelocity(ctx, tx.UserID)
		if vel > 10 {
			isVelocityFraud = true
		}
	}

	alert, err := d.aiClient.Analyze(ctx, tx, user)
	if err != nil {
		slog.Error("AI Analysis failed", "err", err)
		alert = domain.FraudAlert{
			IsBlocked: false,
			Reason:    "AI Service Error - FailSafe Active",
		}
	}

	finalBlocked := isVelocityFraud || alert.IsBlocked

	event := &domain.FraudEvent{
		TransactionID: tx.ID,
		UserID:        tx.UserID,
		Merchant:      tx.Merchant,
		Amount:        tx.Amount,
		Location:      tx.Location,
		IsBlocked:     finalBlocked,
		AIReason:      alert.Reason,
	}

	if err := d.repo.SaveFraudEvent(ctx, event); err != nil {
		slog.Error("Failed to save fraud event", "err", err)
	}

	if d.cache != nil {
		_ = d.cache.IncrementVelocity(ctx, tx.UserID, tx.Location)
	}

	outAlert := domain.FraudAlert{
		TransactionID: tx.ID,
		IsBlocked:     finalBlocked,
		Reason:        alert.Reason,
		Amount:        tx.Amount,
		Location:      tx.Location,
		Merchant:      tx.Merchant,
	}

	if err := d.publisher.Publish(ctx, outAlert); err != nil {
		slog.Error("Failed to publish alert to kafka", "err", err)
		return err
	}

	return nil
}
