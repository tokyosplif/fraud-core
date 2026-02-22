package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/tokyosplif/fraud-core/internal/domain"
)

type SimulatorPublisher interface {
	Send(ctx context.Context, tx domain.Transaction) error
}

type Simulator struct {
	publisher SimulatorPublisher
}

func NewSimulator(s SimulatorPublisher) *Simulator {
	return &Simulator{publisher: s}
}

func (s *Simulator) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tx := s.generateTransaction()

			if err := s.publisher.Send(ctx, tx); err != nil {
				return
			}
		}
	}
}

func (s *Simulator) generateTransaction() domain.Transaction {
	personas := []string{"user-1", "user-2", "user-3"}
	userID := personas[rand.Intn(len(personas))]

	var amount float64
	var merchant, location string

	switch userID {
	case "user-1":
		amount = float64(rand.Intn(40000) + 5000)
		merchant = "Premium Apple Reseller"
		location = "Kyiv, Ukraine"
	case "user-2":
		amount = float64(rand.Intn(3000) + 100)
		merchant = "Binance P2P Exchange"
		location = "Singapore"
	case "user-3":
		amount = float64(rand.Intn(800) + 20)
		merchant = "Local Supermarket"
		location = "Lviv, Ukraine"
	}

	if rand.Intn(100) < 5 {
		amount = 99999
		location = "Lagos, Nigeria"
		merchant = "Unknown Global Store"
	}

	return domain.Transaction{
		ID:        fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		UserID:    userID,
		Amount:    amount,
		Currency:  "USD",
		Merchant:  merchant,
		Location:  location,
		IP:        fmt.Sprintf("192.168.1.%d", rand.Intn(254)+1),
		Timestamp: time.Now(),
	}
}
