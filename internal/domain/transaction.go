package domain

import "time"

type Transaction struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Merchant  string    `json:"merchant"`
	Location  string    `json:"location"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}
