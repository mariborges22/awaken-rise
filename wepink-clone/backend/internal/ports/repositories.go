package ports

import (
	"context"
	"github.com/wepink-clone/backend/internal/domain/entity"
)

type OrderRepository interface {
	Save(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, id string) (*entity.Order, error)
}

type PaymentRepository interface {
	Save(ctx context.Context, payment *entity.Payment) error
	FindByID(ctx context.Context, id string) (*entity.Payment, error)
	FindByOrderID(ctx context.Context, orderID string) ([]*entity.Payment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*entity.Payment, error)
}

type TransactionManager interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
