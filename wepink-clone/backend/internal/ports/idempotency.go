package ports

import (
	"context"
	"time"
)

type IdempotencyStore interface {
	// Check checks if a key exists and returns its value if present.
	// It's used to avoid re-processing the same request.
	Get(ctx context.Context, key string) (interface{}, bool, error)
	
	// Set stores the result of an operation for a given key with an expiration time.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
