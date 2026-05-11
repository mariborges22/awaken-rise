package entity

import (
	"errors"
	"time"
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
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	ID        string      `json:"id"`
	TenantID  string      `json:"tenant_id"`
	Status    OrderStatus `json:"status"`
	Items     []OrderItem `json:"items"`
	Total     float64     `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func NewOrder(id string, tenantID string, items []OrderItem) *Order {
	var total float64
	for _, item := range items {
		if item.Price < 0 || item.Quantity <= 0 {
			continue // Ou poderíamos retornar um erro aqui
		}
		total += item.Price * float64(item.Quantity)
	}

	return &Order{
		ID:        id,
		TenantID:  tenantID,
		Status:    OrderPending,
		Items:     items,
		Total:     total,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (o *Order) RecalculateTotal() {
	var total float64
	for _, item := range o.Items {
		if item.Price < 0 || item.Quantity <= 0 {
			continue
		}
		total += item.Price * float64(item.Quantity)
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
