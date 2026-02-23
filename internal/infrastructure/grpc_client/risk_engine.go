package grpc_client

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	requestTimeout = 10 * time.Second
	retryAttempts  = 3
	retryDelay     = 200 * time.Millisecond
)

type RiskClient struct {
	client pb.RiskEngineServiceClient
	conn   *grpc.ClientConn
}

func NewRiskClient(addr string) (*RiskClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RiskClient{
		client: pb.NewRiskEngineServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *RiskClient) Analyze(ctx context.Context, tx domain.Transaction, user domain.User) (domain.FraudAlert, error) {
	gCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	userContext := fmt.Sprintf(
		"risk_score:%d,max_tx:%.2f,avg_tx:%.2f",
		user.RiskScore, user.MaxTx, user.AvgTx,
	)

	req := &pb.AnalyzeRequest{
		TransactionId:      tx.ID,
		UserId:             tx.UserID,
		Amount:             tx.Amount,
		Merchant:           tx.Merchant,
		Location:           tx.Location,
		UserProfileContext: userContext,
	}

	var resp *pb.AnalyzeResponse
	err := retry.Do(
		func() error {
			var err error
			resp, err = c.client.AnalyzeTransaction(gCtx, req)
			return err
		},
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
	)

	if err != nil {
		return domain.FraudAlert{}, fmt.Errorf("remote ai-risk-engine failure: %w", err)
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

func (c *RiskClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
