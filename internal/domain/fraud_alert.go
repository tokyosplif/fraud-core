package domain

type FraudAlert struct {
	TransactionID string  `json:"transaction_id"`
	Reason        string  `json:"reason"`
	AIPushMessage string  `json:"ai_push_msg"`
	IsBlocked     bool    `json:"is_blocked"`
	Amount        float64 `json:"amount"`
	Location      string  `json:"location"`
	Merchant      string  `json:"merchant"`
}
