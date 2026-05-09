package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
	"github.com/google/uuid"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/pkg/logger"
	"github.com/awaken-rise/backend/internal/ports"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrPaymentFailed = errors.New("payment processing failed")
)

type PaymentUseCase struct {
	paymentRepo      ports.PaymentRepository
	orderRepo        ports.OrderRepository
	tenantRepo       ports.TenantRepository
	idempotencyStore ports.IdempotencyStore
	txManager        ports.TransactionManager
	publisher        ports.EventPublisher
	gateway          ports.PaymentGateway
}

func NewPaymentUseCase(
	paymentRepo ports.PaymentRepository,
	orderRepo ports.OrderRepository,
	tenantRepo ports.TenantRepository,
	idempotencyStore ports.IdempotencyStore,
	txManager ports.TransactionManager,
	publisher ports.EventPublisher,
	gateway ports.PaymentGateway,
) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo:      paymentRepo,
		orderRepo:        orderRepo,
		tenantRepo:       tenantRepo,
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
	PaymentMethod  string
	CardToken      string
	BuyerEmail     string
}

func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, input ProcessPaymentInput) (*entity.Payment, error) {
	correlationID, _ := ctx.Value(logger.CorrelationIDKey).(string)
	
	logger.Info(ctx, "Starting payment process", 
		"order_id", input.OrderID, 
		"idempotency_key", input.IdempotencyKey,
		"correlation_id", correlationID)

	// 1. Idempotency Check - Phase 1: Redis (Fast)
	if input.IdempotencyKey != "" {
		if val, exists, err := uc.idempotencyStore.Get(ctx, input.IdempotencyKey); err == nil && exists {
			logger.Info(ctx, "Idempotency hit in Redis", "key", input.IdempotencyKey)
			if payment, ok := val.(*entity.Payment); ok {
				return payment, nil
			}
		}

		// 2. Idempotency Check - Phase 2: MySQL (Source of Truth)
		if payment, err := uc.paymentRepo.FindByIdempotencyKey(ctx, input.IdempotencyKey); err == nil && payment != nil {
			logger.Info(ctx, "Idempotency hit in MySQL", "key", input.IdempotencyKey)
			_ = uc.idempotencyStore.Set(ctx, input.IdempotencyKey, payment, 24*time.Hour)
			return payment, nil
		}
	}

	// 3. Validate Order and Recalculate Total (Security)
	order, err := uc.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil || order == nil {
		logger.Error(ctx, "Order not found", "order_id", input.OrderID)
		return nil, ErrOrderNotFound
	}
	if order.Status == entity.OrderCancelled {
		return nil, errors.New("cannot pay for a cancelled order")
	}
	
	order.RecalculateTotal()

	// 4. Get Tenant Config (Multi-tenant)
	tenant, err := uc.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil || tenant == nil {
		logger.Error(ctx, "Tenant config not found", "tenant_id", order.TenantID)
		return nil, fmt.Errorf("tenant configuration not found for ID: %s", order.TenantID)
	}

	// 5. Create and Process Payment
	payment := entity.NewPayment(input.PaymentID, input.OrderID, order.Total, input.IdempotencyKey)
	
	logger.Info(ctx, "Calling payment gateway", "amount", order.Total, "method", input.PaymentMethod)

	resp, err := uc.gateway.Process(ctx, ports.PaymentRequest{
		Amount:        order.Total,
		Description:   fmt.Sprintf("Pedido #%s", order.ID),
		PaymentMethod: input.PaymentMethod,
		Token:         input.CardToken,
		Email:         input.BuyerEmail,
		TenantToken:   tenant.MPAccessToken,
	})

	if err != nil {
		logger.Error(ctx, "Gateway error", "error", err)
		payment.Fail()
	} else if resp.Success {
		logger.Info(ctx, "Payment approved by gateway", "transaction_id", resp.TransactionID)
		_ = payment.Approve()
		payment.TransactionID = resp.TransactionID
	} else {
		logger.Warn(ctx, "Payment rejected by gateway", "status", resp.Status, "error", resp.ErrorMessage)
		payment.Fail()
	}

	// 6. Persistence and State Consistency (Atomic)
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
		logger.Error(ctx, "Transaction failed", "error", err)
		return nil, err
	}

	// 7. Store Idempotency Result
	if input.IdempotencyKey != "" {
		_ = uc.idempotencyStore.Set(ctx, input.IdempotencyKey, payment, 24*time.Hour)
	}

	// 8. Publish Response Events with CorrelationID
	event := entity.Event{
		ID:            uuid.New().String(),
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
	}

	if payment.Status == entity.PaymentApproved {
		event.Type = "payment.approved"
		event.Payload = entity.PaymentApprovedPayload{
			PaymentID:      payment.ID,
			OrderID:        payment.OrderID,
			TransactionID:  payment.TransactionID,
			Amount:         payment.Amount,
			IdempotencyKey: input.IdempotencyKey,
		}
	} else {
		event.Type = "payment.failed"
		event.Payload = entity.PaymentFailedPayload{
			PaymentID: payment.ID,
			OrderID:   payment.OrderID,
			Reason:    "payment rejected",
		}
	}

	// Publish with a simple retry logic
	go func(ev entity.Event) {
		for i := 0; i < 3; i++ {
			if err := uc.publisher.Publish(context.Background(), "payments_exchange", ev.Type, ev); err == nil {
				return
			}
			time.Sleep(time.Second * time.Duration(i+1))
		}
		logger.Error(ctx, "Failed to publish event after retries", "event_id", ev.ID)
	}(event)

	return payment, nil
}

type WebhookPayload struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (uc *PaymentUseCase) HandleWebhook(ctx context.Context, payload WebhookPayload) error {
	correlationID, _ := ctx.Value(logger.CorrelationIDKey).(string)
	
	logger.Info(ctx, "Received webhook", "action", payload.Action, "type", payload.Type, "transaction_id", payload.Data.ID, "correlation_id", correlationID)

	if payload.Type != "payment" {
		return nil // Ignore non-payment webhooks
	}

	// For security, we should query Mercado Pago API to get the real status of payload.Data.ID
	// Here we simulate the status update for demonstration.
	// In a real scenario, you'd fetch the Payment by TransactionID:
	// payment, err := uc.paymentRepo.FindByTransactionID(ctx, payload.Data.ID)
	
	// Assuming payment is found and we verify the status is now "approved"
	logger.Info(ctx, "Webhook processing complete (Simulated)", "transaction_id", payload.Data.ID)
	return nil
}

