package grpc_client

import (
	"context"
	"fmt"
	"time"

	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RiskClient struct {
	client pb.RiskEngineServiceClient
	conn   *grpc.ClientConn
}

func NewRiskClient(addr string) (*RiskClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &RiskClient{
		client: pb.NewRiskEngineServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *RiskClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *RiskClient) Analyze(ctx context.Context, tx domain.Transaction, user domain.User) (domain.FraudAlert, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	userContext := fmt.Sprintf("risk_score:%d,is_banned:%t", user.RiskScore, user.IsBanned)

	req := &pb.AnalyzeRequest{
		TransactionId:      tx.ID,
		UserId:             tx.UserID,
		Amount:             tx.Amount,
		Merchant:           tx.Merchant,
		Location:           tx.Location,
		UserProfileContext: userContext,
	}

	resp, err := c.client.AnalyzeTransaction(ctx, req)
	if err != nil {
		return domain.FraudAlert{}, err
	}

	return domain.FraudAlert{
		TransactionID: tx.ID,
		Reason:        resp.GetReason(),
		AIPushMessage: resp.GetAiPushMsg(),
		IsBlocked:     resp.GetIsBlocked(),
		Amount:        tx.Amount,
		Location:      tx.Location,
		Merchant:      tx.Merchant,
	}, nil
}
