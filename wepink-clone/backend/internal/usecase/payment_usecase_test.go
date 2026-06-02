package usecase

import (
	"context"
	"testing"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/service"
)

type mockOrderRepo struct {
	orders map[string]*entity.Order
}

func (m *mockOrderRepo) Save(ctx context.Context, o *entity.Order) error {
	m.orders[o.ID] = o
	return nil
}

func (m *mockOrderRepo) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	return m.orders[id], nil
}

func TestPaymentUseCase_PlanEnforcement(t *testing.T) {
	orderRepo := &mockOrderRepo{orders: make(map[string]*entity.Order)}
	tenantRepo := &mockTenantRepo{tenants: make(map[string]*entity.Tenant)}
	
	// Criar o UseCase com mocks mínimos (apenas o necessário para testar o enforcement)
	idempotency := service.NewIdempotencyService(nil, nil)
	uc := NewPaymentUseCase(nil, orderRepo, tenantRepo, nil, nil, nil, nil, idempotency, nil)

	t.Run("Block Sale for Pending Tenant", func(t *testing.T) {
		tenantID := "pending-shop"
		tenantRepo.tenants[tenantID] = &entity.Tenant{
			ID:                 tenantID,
			Status:             "active",
			VerificationStatus: entity.VerificationPending,
		}

		orderID := "order-1"
		orderRepo.orders[orderID] = &entity.Order{ID: orderID, TenantID: tenantID, Status: entity.OrderPending}

		input := ProcessPaymentInput{OrderID: orderID}
		_, _, err := uc.ProcessPayment(context.Background(), input)

		if err == nil || err.Error() != "subscription limit reached or account not verified" {
			t.Errorf("expected verification error, got %v", err)
		}
	})

	t.Run("Block Sale for Starter Limit Reached", func(t *testing.T) {
		tenantID := "starter-full"
		tenantRepo.tenants[tenantID] = &entity.Tenant{
			ID:                 tenantID,
			Status:             "active",
			VerificationStatus: entity.VerificationApproved,
			Plan:               entity.PlanStarter,
			MonthlyUsageCount:  100, // Limite do starter
		}

		orderID := "order-2"
		orderRepo.orders[orderID] = &entity.Order{ID: orderID, TenantID: tenantID, Status: entity.OrderPending}

		input := ProcessPaymentInput{OrderID: orderID}
		_, _, err := uc.ProcessPayment(context.Background(), input)

		if err == nil || err.Error() != "subscription limit reached or account not verified" {
			t.Errorf("expected plan limit error, got %v", err)
		}
	})
}
