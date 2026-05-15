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
	OrderID  string       `json:"order_id"`
	TenantID string       `json:"tenant_id"`
	Total    kernel.Money `json:"total"`
}

type PaymentApprovedPayload struct {
	PaymentID      string       `json:"payment_id"`
	OrderID        string       `json:"order_id"`
	TransactionID  string       `json:"transaction_id"`
	Amount         kernel.Money `json:"amount"`
	IdempotencyKey string       `json:"idempotency_key"`
}

type PaymentFailedPayload struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
}
