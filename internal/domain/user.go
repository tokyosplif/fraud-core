package domain

import "time"

type User struct {
	ID        string       `gorm:"primaryKey"`
	Email     string       `gorm:"size:255;default:'unknown@mail.com'"`
	RiskScore int          `gorm:"default:10"`
	IsBanned  bool         `gorm:"default:false"`
	MaxTx     float64      `gorm:"-"`
	AvgTx     float64      `gorm:"-"`
	Events    []FraudEvent `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
