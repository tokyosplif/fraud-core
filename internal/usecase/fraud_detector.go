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
	GetUserStats(ctx context.Context, userID string) (float64, float64, error)
}

type CacheRepository interface {
	GetVelocity(ctx context.Context, userID string) (int, error)
	IncrementVelocity(ctx context.Context, userID, location string) error
	GetUserStats(ctx context.Context, userID string) (float64, float64, error)
	SetUserStats(ctx context.Context, userID string, maxAmount, avgAmount float64) error
	GetRiskCache(ctx context.Context, userID, merchant string) (*domain.FraudAlert, error)
	SetRiskCache(ctx context.Context, userID, merchant string, alert domain.FraudAlert) error
}

type FraudPublisher interface {
	Publish(ctx context.Context, alert domain.FraudAlert) error
}

type FraudDetector struct {
	aiClient      AIClient
	repo          Repository
	cache         CacheRepository
	publisher     FraudPublisher
	velocityLimit int
}

func NewFraudDetector(ai AIClient, r Repository, c CacheRepository, p FraudPublisher) *FraudDetector {
	return &FraudDetector{
		aiClient:      ai,
		repo:          r,
		cache:         c,
		publisher:     p,
		velocityLimit: 10,
	}
}

func (d *FraudDetector) Detect(ctx context.Context, tx domain.Transaction) error {
	user, _ := d.repo.GetUserByID(ctx, tx.UserID)
	if user == nil {
		user = &domain.User{ID: tx.UserID, RiskScore: 15}
		_ = d.repo.CreateUser(ctx, user)
	}

	maxTx, avgTx, _ := d.cache.GetUserStats(ctx, tx.UserID)
	if maxTx == 0 && avgTx == 0 {
		maxTx, avgTx, _ = d.repo.GetUserStats(ctx, tx.UserID)
		_ = d.cache.SetUserStats(ctx, tx.UserID, maxTx, avgTx)
	}
	user.MaxTx = maxTx
	user.AvgTx = avgTx

	isVelocityFraud := false
	vel, _ := d.cache.GetVelocity(ctx, tx.UserID)
	if vel > d.velocityLimit {
		isVelocityFraud = true
	}

	var alert domain.FraudAlert
	cachedAlert, _ := d.cache.GetRiskCache(ctx, tx.UserID, tx.Merchant)

	if cachedAlert != nil {
		alert = *cachedAlert
	} else {
		var err error
		alert, err = d.aiClient.Analyze(ctx, tx, *user)
		if err != nil {
			slog.Error("AI Analysis failed", "err", err)
			alert = domain.FraudAlert{
				IsBlocked: false,
				Reason:    "AI Service Error - FailSafe Active",
			}
		} else {
			_ = d.cache.SetRiskCache(ctx, tx.UserID, tx.Merchant, alert)
		}
	}

	finalBlocked := isVelocityFraud || alert.IsBlocked
	reason := alert.Reason

	if isVelocityFraud {
		if alert.IsBlocked {
			reason = "[Velocity Block] + " + alert.Reason
		} else {
			reason = "[Velocity Block] User exceeded transaction frequency limit"
		}
	}

	event := &domain.FraudEvent{
		TransactionID: tx.ID,
		UserID:        tx.UserID,
		Merchant:      tx.Merchant,
		Amount:        tx.Amount,
		Location:      tx.Location,
		IsBlocked:     finalBlocked,
		AIReason:      reason,
	}
	if err := d.repo.SaveFraudEvent(ctx, event); err != nil {
		slog.Error("Failed to save fraud event", "err", err)
	}

	_ = d.cache.IncrementVelocity(ctx, tx.UserID, tx.Location)

	outAlert := domain.FraudAlert{
		TransactionID: tx.ID,
		IsBlocked:     finalBlocked,
		Reason:        reason,
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
