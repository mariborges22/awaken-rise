package entity

import (
	"errors"
	"time"

	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "PENDING"
	PaymentApproved PaymentStatus = "APPROVED"
	PaymentFailed   PaymentStatus = "FAILED"
)

var (
	ErrInvalidPaymentTransition = errors.New("invalid payment status transition")
	ErrPaymentAlreadyApproved   = errors.New("payment is already approved")
)

type Payment struct {
	kernel.AggregateRoot
	ID             string        `json:"id"`
	OrderID        string        `json:"order_id"`
	TransactionID  string        `json:"transaction_id"`
	Amount         kernel.Money  `json:"amount"`
	Status         PaymentStatus `json:"status"`
	IdempotencyKey string        `json:"idempotency_key"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func NewPayment(id, orderID string, amount kernel.Money, idempotencyKey string) *Payment {
	return &Payment{
		ID:             id,
		OrderID:        orderID,
		Amount:         amount,
		Status:         PaymentPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (p *Payment) Approve() error {
	if p.Status == PaymentApproved {
		return nil // Idempotent
	}
	if p.Status == PaymentFailed {
		return ErrInvalidPaymentTransition
	}
	p.Status = PaymentApproved
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Payment) Fail() {
	if p.Status == PaymentApproved {
		return // Cannot fail an approved payment in this context
	}
	p.Status = PaymentFailed
	p.UpdatedAt = time.Now()
}
