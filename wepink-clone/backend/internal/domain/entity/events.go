package entity

import "time"

type Event struct {
	ID        string      `json:"event_id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type OrderCreatedPayload struct {
	OrderID string  `json:"order_id"`
	Total   float64 `json:"total"`
}

type PaymentApprovedPayload struct {
	PaymentID string  `json:"payment_id"`
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
}

type PaymentFailedPayload struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
}
