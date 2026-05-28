package ports

import (
	"context"
	"github.com/awaken-rise/backend/internal/domain/entity"
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
	FindByTransactionID(ctx context.Context, transactionID string) (*entity.Payment, error)
}

type TenantRepository interface {
	FindByID(ctx context.Context, id string) (*entity.Tenant, error)
	Save(ctx context.Context, tenant *entity.Tenant) error
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Save(ctx context.Context, user *entity.User) error
}

type ProductRepository interface {
	FindByID(ctx context.Context, id string) (*entity.Product, error)
	FindByIDs(ctx context.Context, ids []string) ([]*entity.Product, error)
	Save(ctx context.Context, product *entity.Product) error
	SaveBatch(ctx context.Context, products []*entity.Product) error
	ListByTenant(ctx context.Context, tenantID string) ([]*entity.Product, error)
}

type TransactionManager interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// ProductCacheStore define o contrato para o cache de vitrine de produtos.
// O adaptador Redis implementa essa interface; o ProductUseCase depende apenas dela.
type ProductCacheStore interface {
	Get(ctx context.Context, tenantID string) ([]*entity.Product, bool, error)
	Set(ctx context.Context, tenantID string, products []*entity.Product) error
	Invalidate(ctx context.Context, tenantID string) error
}
