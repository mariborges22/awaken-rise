package usecase

import (
	"context"
	"fmt"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type OrderUseCase struct {
	orderRepo  ports.OrderRepository
	dispatcher ports.EventDispatcher
}

func NewOrderUseCase(orderRepo ports.OrderRepository, dispatcher ports.EventDispatcher) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:  orderRepo,
		dispatcher: dispatcher,
	}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, id string, items []entity.OrderItem) (*entity.Order, error) {
	tenantID, _ := kernel.GetTenantID(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	order := entity.NewOrder(id, tenantID, items)
	
	if err := uc.orderRepo.Save(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Dispatch collected events
	if len(order.Events()) > 0 {
		_ = uc.dispatcher.Dispatch(ctx, order.Events())
		order.ClearEvents()
	}
	
	return order, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*entity.Order, error) {
	order, err := uc.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}
	return order, nil
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) error {
	order, err := uc.orderRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}
	
	if err := order.Cancel(); err != nil {
		return err
	}
	
	return uc.orderRepo.Save(ctx, order)
}
