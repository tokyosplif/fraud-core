package grpc_client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/redis/go-redis/v9"
	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RiskClient struct {
	client pb.RiskEngineServiceClient
	conn   *grpc.ClientConn
	redis  *redis.Client
}

func NewRiskClient(addr string, rdb *redis.Client) (*RiskClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RiskClient{
		client: pb.NewRiskEngineServiceClient(conn),
		conn:   conn,
		redis:  rdb,
	}, nil
}

func (c *RiskClient) Analyze(ctx context.Context, tx domain.Transaction, user domain.User) (domain.FraudAlert, error) {
	cacheKey := fmt.Sprintf("ai_risk:%s:%s", tx.UserID, tx.Merchant)

	if val, err := c.redis.Get(ctx, cacheKey).Result(); err == nil {
		var cachedAlert domain.FraudAlert
		if err := json.Unmarshal([]byte(val), &cachedAlert); err == nil {
			return cachedAlert, nil
		}
	}

	resp, err := c.fetchRemoteAnalysis(ctx, tx, user)
	if err != nil {
		return domain.FraudAlert{}, err
	}

	alert := domain.FraudAlert{
		TransactionID: tx.ID,
		Reason:        resp.GetReason(),
		AIPushMessage: resp.GetAiPushMsg(),
		IsBlocked:     resp.GetIsBlocked(),
		Amount:        tx.Amount,
		Location:      tx.Location,
		Merchant:      tx.Merchant,
	}

	if data, err := json.Marshal(alert); err == nil {
		c.redis.Set(ctx, cacheKey, data, time.Minute*5)
	}
	return alert, nil
}

func (c *RiskClient) fetchRemoteAnalysis(ctx context.Context, tx domain.Transaction, user domain.User) (*pb.AnalyzeResponse, error) {
	gCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		retry.Attempts(3),
		retry.Delay(200*time.Millisecond),
	)

	if err != nil {
		return nil, fmt.Errorf("remote ai-risk-engine failure: %w", err)
	}
	return resp, nil
}

func (c *RiskClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
