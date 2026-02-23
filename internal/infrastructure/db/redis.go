package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokyosplif/fraud-core/internal/domain"
)

const (
	velocityTTL   = time.Minute
	statsTTL      = 10 * time.Minute
	aiRiskTTL     = 5 * time.Minute
	scanBatchSize = 10
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func (r *RedisRepository) GetVelocity(ctx context.Context, userID string) (int, error) {
	pattern := fmt.Sprintf("velocity:%s:*", userID)
	var cursor uint64
	count := 0

	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, scanBatchSize).Result()
		if err != nil {
			return 0, err
		}
		count += len(keys)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return count, nil
}

func (r *RedisRepository) IncrementVelocity(ctx context.Context, userID, location string) error {
	key := fmt.Sprintf("velocity:%s:%s", userID, location)
	pipe := r.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, velocityTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) GetUserStats(ctx context.Context, userID string) (float64, float64, error) {
	key := fmt.Sprintf("user_stats:%s", userID)
	val, err := r.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, err
	}

	var maxAmount, avgAmount float64
	_, err = fmt.Sscanf(val, "%f|%f", &maxAmount, &avgAmount)
	return maxAmount, avgAmount, err
}

func (r *RedisRepository) SetUserStats(ctx context.Context, userID string, maxAmount, avgAmount float64) error {
	key := fmt.Sprintf("user_stats:%s", userID)
	return r.rdb.Set(ctx, key, fmt.Sprintf("%.2f|%.2f", maxAmount, avgAmount), statsTTL).Err()
}

func (r *RedisRepository) GetRiskCache(ctx context.Context, userID, merchant string) (*domain.FraudAlert, error) {
	key := fmt.Sprintf("ai_risk:%s:%s", userID, merchant)
	val, err := r.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var alert domain.FraudAlert
	if err := json.Unmarshal([]byte(val), &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *RedisRepository) SetRiskCache(ctx context.Context, userID, merchant string, alert domain.FraudAlert) error {
	key := fmt.Sprintf("ai_risk:%s:%s", userID, merchant)
	data, _ := json.Marshal(alert)
	return r.rdb.Set(ctx, key, data, aiRiskTTL).Err()
}
