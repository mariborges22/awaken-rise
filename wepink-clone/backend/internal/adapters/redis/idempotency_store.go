package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyStore struct {
	client *redis.Client
}

func NewIdempotencyStore(client *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{client: client}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (interface{}, bool, error) {
	val, err := s.client.Get(ctx, "idempotency:"+key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get from redis: %w", err)
	}

	// Note: In a real scenario, we might need a way to deserialize into specific types.
	// For now, we'll return the raw string or a generic map if it's JSON.
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return val, true, nil // Return as string if not JSON
	}

	return result, true, nil
}

func (s *IdempotencyStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = s.client.Set(ctx, "idempotency:"+key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set in redis: %w", err)
	}

	return nil
}
