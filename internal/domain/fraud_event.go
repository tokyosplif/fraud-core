package domain

import "time"

type FraudEvent struct {
	ID            uint    `gorm:"primaryKey"`
	TransactionID string  `gorm:"uniqueIndex;not null;size:100"`
	UserID        string  `gorm:"index;not null;size:100"`
	Merchant      string  `gorm:"size:255"`
	Amount        float64 `gorm:"type:decimal(10,2)"`
	Location      string  `gorm:"size:255"`
	IsBlocked     bool
	AIReason      string    `gorm:"type:text"`
	AIPushMsg     string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}
