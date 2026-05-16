package entity

import (
	"errors"
	"time"

	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type OrderStatus string

const (
	OrderPending   OrderStatus = "PENDING"
	OrderConfirmed OrderStatus = "CONFIRMED"
	OrderCancelled OrderStatus = "CANCELLED"
)

var (
	ErrInvalidOrderTransition = errors.New("invalid order status transition")
	ErrOrderAlreadyConfirmed  = errors.New("order is already confirmed")
	ErrOrderAlreadyCancelled  = errors.New("order is already cancelled")
)

type OrderItem struct {
	ProductID string       `json:"product_id"`
	Quantity  int          `json:"quantity"`
	Price     kernel.Money `json:"price"`
}

type Order struct {
	kernel.AggregateRoot
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Status    OrderStatus  `json:"status"`
	Items     []OrderItem  `json:"items"`
	Total     kernel.Money `json:"total"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func NewOrder(id string, tenantID string, items []OrderItem) *Order {
	total := kernel.NewBRL(0)
	for _, item := range items {
		// Multiplicação de centavos
		itemTotal := item.Price.Multiply(int64(item.Quantity))
		total, _ = total.Add(itemTotal)
	}

	order := &Order{
		ID:        id,
		TenantID:  tenantID,
		Status:    OrderPending,
		Items:     items,
		Total:     total,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	eventPayload := OrderCreatedPayload{
		BaseDomainEvent: kernel.NewBaseDomainEvent(),
		OrderID:         id,
		TenantID:        tenantID,
		Total:           total,
	}
	order.AddEvent(eventPayload)

	return order
}

func (o *Order) RecalculateTotal() {
	total := kernel.NewBRL(0)
	for _, item := range o.Items {
		itemTotal := item.Price.Multiply(int64(item.Quantity))
		total, _ = total.Add(itemTotal)
	}
	o.Total = total
}

func (o *Order) Confirm() error {
	if o.Status == OrderConfirmed {
		return nil // Idempotent
	}
	if o.Status == OrderCancelled {
		return ErrOrderAlreadyCancelled
	}
	o.Status = OrderConfirmed
	o.UpdatedAt = time.Now()
	
	// Exemplo de registro de evento de domínio
	// o.AddEvent(NewOrderConfirmedEvent(o.ID))
	
	return nil
}

func (o *Order) Cancel() error {
	if o.Status == OrderCancelled {
		return nil // Idempotent
	}
	if o.Status == OrderConfirmed {
		return ErrOrderAlreadyConfirmed
	}
	o.Status = OrderCancelled
	o.UpdatedAt = time.Now()
	return nil
}
