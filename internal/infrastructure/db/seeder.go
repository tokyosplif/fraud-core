package db

import (
	"log/slog"

	"github.com/tokyosplif/fraud-core/internal/domain"
	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB) error {
	users := []domain.User{
		{
			ID:        "user-1",
			Email:     "ceo@apple.com",
			RiskScore: 0,
			IsBanned:  false,
		},
		{
			ID:        "user-2",
			Email:     "hacker99@darknet.com",
			RiskScore: 85,
			IsBanned:  false,
		},
		{
			ID:        "user-3",
			Email:     "john.doe@gmail.com",
			RiskScore: 15,
			IsBanned:  false,
		},
	}

	for _, u := range users {
		if err := db.FirstOrCreate(&u, domain.User{ID: u.ID}).Error; err != nil {
			slog.Error("Failed to seed user", "user_id", u.ID, "err", err)
			return err
		}
	}

	return nil
}
