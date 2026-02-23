package db

import (
	"context"
	"errors"

	"github.com/tokyosplif/fraud-core/internal/domain"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresRepository) SaveFraudEvent(ctx context.Context, event *domain.FraudEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *PostgresRepository) GetUserStats(ctx context.Context, userID string) (float64, float64, error) {
	var stats struct {
		MaxAmount float64 `gorm:"column:max_amount"`
		AvgAmount float64 `gorm:"column:avg_amount"`
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(MAX(amount),0) as max_amount, COALESCE(AVG(amount),0) as avg_amount 
		FROM fraud_events WHERE user_id = ?`, userID).Scan(&stats).Error

	return stats.MaxAmount, stats.AvgAmount, err
}
