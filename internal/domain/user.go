package domain

import "time"

type User struct {
	ID        string       `gorm:"primaryKey"`
	Email     string       `gorm:"size:255;default:'unknown@mail.com'"` // Додаємо size для рядків
	RiskScore int          `gorm:"default:10"`
	IsBanned  bool         `gorm:"default:false"`
	Events    []FraudEvent `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
