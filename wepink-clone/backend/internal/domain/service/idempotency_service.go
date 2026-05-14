package service

import (
	"context"
	"time"
	"log/slog"

	"github.com/awaken-rise/backend/internal/ports"
)

type IdempotencyService struct {
	store ports.IdempotencyStore
	repo  ports.PaymentRepository // Poderia ser um repositório genérico no futuro
}

func NewIdempotencyService(store ports.IdempotencyStore, repo ports.PaymentRepository) *IdempotencyService {
	return &IdempotencyService{
		store: store,
		repo:  repo,
	}
}

func (s *IdempotencyService) GetPayment(ctx context.Context, key string) (interface{}, bool, error) {
	if key == "" {
		return nil, false, nil
	}

	// 1. Check Redis (Fast)
	val, exists, err := s.store.Get(ctx, key)
	if err == nil && exists {
		slog.Debug("Idempotency hit in cache", "key", key)
		return val, true, nil
	}

	// 2. Check Database (Source of Truth)
	payment, err := s.repo.FindByIdempotencyKey(ctx, key)
	if err == nil && payment != nil {
		slog.Debug("Idempotency hit in database", "key", key)
		// Update cache
		_ = s.store.Set(ctx, key, payment, 24*time.Hour)
		return payment, true, nil
	}

	return nil, false, nil
}

func (s *IdempotencyService) Save(ctx context.Context, key string, value interface{}) error {
	if key == "" {
		return nil
	}
	return s.store.Set(ctx, key, value, 24*time.Hour)
}
