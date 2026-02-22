package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func (r *RedisRepository) GetLastLocation(ctx context.Context, userID string) (string, error) {
	locKey := fmt.Sprintf("location:%s", userID)
	loc, err := r.rdb.Get(ctx, locKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return loc, err
}

func (r *RedisRepository) GetVelocity(ctx context.Context, userID string) (int, error) {
	pattern := fmt.Sprintf("velocity:%s:*", userID)
	var cursor uint64
	count := 0

	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 10).Result()
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

func (r *RedisRepository) UpdateLocation(ctx context.Context, userID, location string) error {
	locKey := fmt.Sprintf("location:%s", userID)
	return r.rdb.Set(ctx, locKey, location, 24*time.Hour).Err()
}

func (r *RedisRepository) IncrementVelocity(ctx context.Context, userID, location string) error {
	key := fmt.Sprintf("velocity:%s:%s", userID, location)
	pipe := r.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}
