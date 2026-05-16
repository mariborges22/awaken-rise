package entity

import (
	"time"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type Event struct {
	ID            string      `json:"event_id"`
	CorrelationID string      `json:"correlation_id"`
	Type          string      `json:"type"`
	Timestamp     time.Time   `json:"timestamp"`
	Payload       interface{} `json:"payload"`
}

type OrderCreatedPayload struct {
	kernel.BaseDomainEvent
	OrderID  string       `json:"order_id"`
	TenantID string       `json:"tenant_id"`
	Total    kernel.Money `json:"total"`
}

func (e OrderCreatedPayload) EventName() string {
	return "order.created"
}

type PaymentApprovedPayload struct {
	kernel.BaseDomainEvent
	PaymentID      string       `json:"payment_id"`
	OrderID        string       `json:"order_id"`
	TransactionID  string       `json:"transaction_id"`
	Amount         kernel.Money `json:"amount"`
	IdempotencyKey string       `json:"idempotency_key"`
}

func (e PaymentApprovedPayload) EventName() string {
	return "payment.approved"
}

type PaymentFailedPayload struct {
	kernel.BaseDomainEvent
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
}

func (e PaymentFailedPayload) EventName() string {
	return "payment.failed"
}

