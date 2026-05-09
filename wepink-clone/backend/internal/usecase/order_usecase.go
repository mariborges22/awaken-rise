package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/ports"
)

type OrderUseCase struct {
	orderRepo ports.OrderRepository
	publisher ports.EventPublisher
}

func NewOrderUseCase(orderRepo ports.OrderRepository, publisher ports.EventPublisher) *OrderUseCase {
	return &OrderUseCase{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, id string, tenantID string, items []entity.OrderItem) (*entity.Order, error) {
	order := entity.NewOrder(id, tenantID, items)
	
	if err := uc.orderRepo.Save(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Publish event
	event := entity.Event{
		ID:        uuid.New().String(),
		Type:      "order.created",
		Timestamp: time.Now(),
		Payload: entity.OrderCreatedPayload{
			OrderID: order.ID,
			Total:   order.Total,
		},
	}
	_ = uc.publisher.Publish(ctx, "orders_exchange", "order.created", event)
	
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
