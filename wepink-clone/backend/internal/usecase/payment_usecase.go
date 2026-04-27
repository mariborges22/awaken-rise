package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"github.com/google/uuid"

	"github.com/wepink-clone/backend/internal/domain/entity"
	"github.com/wepink-clone/backend/internal/pkg/logger"
	"github.com/wepink-clone/backend/internal/ports"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrPaymentFailed = errors.New("payment processing failed")
)

type PaymentUseCase struct {
	paymentRepo      ports.PaymentRepository
	orderRepo        ports.OrderRepository
	idempotencyStore ports.IdempotencyStore
	txManager        ports.TransactionManager
	publisher        ports.EventPublisher
	gateway          ports.PaymentGateway
}

func NewPaymentUseCase(
	paymentRepo ports.PaymentRepository,
	orderRepo ports.OrderRepository,
	idempotencyStore ports.IdempotencyStore,
	txManager ports.TransactionManager,
	publisher ports.EventPublisher,
	gateway ports.PaymentGateway,
) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo:      paymentRepo,
		orderRepo:        orderRepo,
		idempotencyStore: idempotencyStore,
		txManager:        txManager,
		publisher:        publisher,
		gateway:          gateway,
	}
}

type ProcessPaymentInput struct {
	PaymentID      string
	OrderID        string
	IdempotencyKey string
}

func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, input ProcessPaymentInput) (*entity.Payment, error) {
	// 1. Idempotency Check - Phase 1: Redis (Fast)
	if input.IdempotencyKey != "" {
		if val, exists, err := uc.idempotencyStore.Get(ctx, input.IdempotencyKey); err == nil && exists {
			// If found in Redis, return immediately
			if payment, ok := val.(*entity.Payment); ok {
				return payment, nil
			}
		}

		// 2. Idempotency Check - Phase 2: MySQL (Source of Truth)
		// Check MySQL in case Redis was purged or it's a retry of a completed operation
		if payment, err := uc.paymentRepo.FindByIdempotencyKey(ctx, input.IdempotencyKey); err == nil && payment != nil {
			// Update Redis for next time
			_ = uc.idempotencyStore.Set(ctx, input.IdempotencyKey, payment, 24*time.Hour)
			return payment, nil
		}
	}

	// 3. Validate Order and Recalculate Total (Security)
	order, err := uc.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.Status == entity.OrderCancelled {
		return nil, errors.New("cannot pay for a cancelled order")
	}
	
	// Recalculate total from items stored in DB to ensure security
	order.RecalculateTotal()

	// 3. Create and Process Payment
	payment := entity.NewPayment(input.PaymentID, input.OrderID, order.Total, input.IdempotencyKey)
	
	// Call external payment gateway (Simulated)
	resp, err := uc.gateway.Process(ctx, order.Total)
	if err != nil {
		logger.Error(ctx, "Gateway error", "error", err)
		payment.Fail()
	} else if resp.Success {
		_ = payment.Approve()
	} else {
		payment.Fail()
	}

	// 4. Persistence and State Consistency (Atomic)
	err = uc.txManager.Execute(ctx, func(ctx context.Context) error {
		if err := uc.paymentRepo.Save(ctx, payment); err != nil {
			return fmt.Errorf("failed to save payment: %w", err)
		}

		if payment.Status == entity.PaymentApproved {
			if err := order.Confirm(); err == nil {
				if err := uc.orderRepo.Save(ctx, order); err != nil {
					return fmt.Errorf("failed to update order: %w", err)
				}
			}
		}
		return nil
	})

	if err != nil {
		// Fallback for race condition: check if it was already created by another thread
		if input.IdempotencyKey != "" {
			if payment, retryErr := uc.paymentRepo.FindByIdempotencyKey(ctx, input.IdempotencyKey); retryErr == nil && payment != nil {
				_ = uc.idempotencyStore.Set(ctx, input.IdempotencyKey, payment, 24*time.Hour)
				return payment, nil
			}
		}
		return nil, err
	}

	// 5. Store Idempotency Result
	if input.IdempotencyKey != "" {
		_ = uc.idempotencyStore.Set(ctx, input.IdempotencyKey, payment, 24*time.Hour)
	}

	// 6. Publish Response Events
	if payment.Status == entity.PaymentApproved {
		_ = uc.publisher.Publish(ctx, "payments_exchange", "payment.approved", entity.Event{
			ID:        uuid.New().String(),
			Type:      "payment.approved",
			Timestamp: time.Now(),
			Payload: entity.PaymentApprovedPayload{
				PaymentID: payment.ID,
				OrderID:   payment.OrderID,
				Amount:    payment.Amount,
			},
		})
	} else {
		_ = uc.publisher.Publish(ctx, "payments_exchange", "payment.failed", entity.Event{
			ID:        uuid.New().String(),
			Type:      "payment.failed",
			Timestamp: time.Now(),
			Payload: entity.PaymentFailedPayload{
				PaymentID: payment.ID,
				OrderID:   payment.OrderID,
				Reason:    "payment rejected",
			},
		})
	}

	return payment, nil
}
