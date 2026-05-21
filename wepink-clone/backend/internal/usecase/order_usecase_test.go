package usecase_test

import (
	"context"
	"testing"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type MockOrderRepo struct {
	SaveFunc     func(ctx context.Context, order *entity.Order) error
	FindByIDFunc func(ctx context.Context, id string) (*entity.Order, error)
}

func (m *MockOrderRepo) Save(ctx context.Context, order *entity.Order) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, order)
	}
	return nil
}
func (m *MockOrderRepo) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

type MockProductRepo struct {
	FindByIDsFunc func(ctx context.Context, ids []string) ([]*entity.Product, error)
	SaveBatchFunc func(ctx context.Context, products []*entity.Product) error
	// Other methods to satisfy interface
	FindByIDFunc     func(ctx context.Context, id string) (*entity.Product, error)
	SaveFunc         func(ctx context.Context, product *entity.Product) error
	ListByTenantFunc func(ctx context.Context, tenantID string) ([]*entity.Product, error)
}

func (m *MockProductRepo) FindByIDs(ctx context.Context, ids []string) ([]*entity.Product, error) {
	if m.FindByIDsFunc != nil {
		return m.FindByIDsFunc(ctx, ids)
	}
	return nil, nil
}
func (m *MockProductRepo) SaveBatch(ctx context.Context, products []*entity.Product) error {
	if m.SaveBatchFunc != nil {
		return m.SaveBatchFunc(ctx, products)
	}
	return nil
}
func (m *MockProductRepo) FindByID(ctx context.Context, id string) (*entity.Product, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockProductRepo) Save(ctx context.Context, product *entity.Product) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, product)
	}
	return nil
}
func (m *MockProductRepo) ListByTenant(ctx context.Context, tenantID string) ([]*entity.Product, error) {
	return nil, nil
}

type MockEventDispatcher struct {
	DispatchFunc func(ctx context.Context, events []kernel.DomainEvent) error
}

func (m *MockEventDispatcher) Dispatch(ctx context.Context, events []kernel.DomainEvent) error {
	if m.DispatchFunc != nil {
		return m.DispatchFunc(ctx, events)
	}
	return nil
}

type MockTxManager struct{}

func (m *MockTxManager) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestOrderUseCase_CreateOrder(t *testing.T) {
	// Tests will be expanded here if needed
}
