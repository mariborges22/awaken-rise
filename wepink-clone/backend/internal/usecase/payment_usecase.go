package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/service"
	"github.com/awaken-rise/backend/internal/pkg/logger"
	"github.com/awaken-rise/backend/internal/pkg/metrics"
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
	txManager        ports.TransactionManager
	dispatcher       ports.EventDispatcher
	gateway          ports.PaymentGateway
	oauth            ports.OAuthProvider
	idempotency      *service.IdempotencyService
	encryption       *service.EncryptionService
}

func NewPaymentUseCase(
	paymentRepo ports.PaymentRepository,
	orderRepo ports.OrderRepository,
	tenantRepo ports.TenantRepository,
	txManager ports.TransactionManager,
	dispatcher ports.EventDispatcher,
	gateway ports.PaymentGateway,
	oauth ports.OAuthProvider,
	idempotency *service.IdempotencyService,
	encryption *service.EncryptionService,
) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo:      paymentRepo,
		orderRepo:        orderRepo,
		tenantRepo:       tenantRepo,
		txManager:        txManager,
		dispatcher:       dispatcher,
		gateway:          gateway,
		oauth:            oauth,
		idempotency:      idempotency,
		encryption:       encryption,
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

func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, input ProcessPaymentInput) (*entity.Payment, *ports.PaymentGatewayResponse, error) {
	correlationID, _ := ctx.Value(logger.CorrelationIDKey).(string)
	
	logger.Info(ctx, "Starting payment process", 
		"order_id", input.OrderID, 
		"idempotency_key", input.IdempotencyKey,
		"correlation_id", correlationID)

	// 1. Idempotency Check (Delegated to Service)
	if input.IdempotencyKey != "" {
		if payment, exists, err := uc.idempotency.GetPayment(ctx, input.IdempotencyKey); err == nil && exists {
			return payment.(*entity.Payment), nil, nil
		}
	}

	// 3. Validate Order and Recalculate Total (Security)
	order, err := uc.orderRepo.FindByID(ctx, input.OrderID)
	if err != nil || order == nil {
		logger.Error(ctx, "Order not found", "order_id", input.OrderID)
		return nil, nil, ErrOrderNotFound
	}
	if order.Status == entity.OrderCancelled {
		return nil, nil, errors.New("cannot pay for a cancelled order")
	}
	
	order.RecalculateTotal()

	// 4. Get Tenant Config (Multi-tenant)
	tenant, err := uc.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil || tenant == nil {
		logger.Error(ctx, "Tenant config not found", "tenant_id", order.TenantID)
		return nil, nil, fmt.Errorf("tenant configuration not found for ID: %s", order.TenantID)
	}

	// 5. Business Rule: Plan & Verification Enforcement
	if !tenant.CanProcessSale() {
		logger.Warn(ctx, "Tenant not allowed to process sale", 
			"tenant_id", tenant.ID, 
			"status", tenant.Status, 
			"verification", tenant.VerificationStatus,
			"usage", tenant.MonthlyUsageCount,
			"plan", tenant.Plan)
		return nil, nil, errors.New("subscription limit reached or account not verified")
	}

	// 5. Create and Process Payment
	payment := entity.NewPayment(input.PaymentID, input.OrderID, order.Total, input.IdempotencyKey)
	
	logger.Info(ctx, "Calling payment gateway", "amount", order.Total, "method", input.PaymentMethod, "provider", tenant.PaymentProvider)

	// Recuperar token real (Criptografado)
	token, _ := uc.getTenantToken(ctx, tenant)

	resp, err := uc.gateway.Process(ctx, ports.PaymentRequest{
		Amount:        order.Total,
		Description:   fmt.Sprintf("Pedido #%s", order.ID),
		PaymentMethod: input.PaymentMethod,
		Token:         input.CardToken,
		Email:         input.BuyerEmail,
		TenantToken:   token,
	})

	if err != nil {
		logger.Error(ctx, "Gateway error", "error", err)
		payment.Fail(err.Error())
	} else if resp.Success {
		logger.Info(ctx, "Payment approved by gateway", "transaction_id", resp.TransactionID)
		_ = payment.Approve()
		payment.TransactionID = resp.TransactionID
	} else {
		logger.Warn(ctx, "Payment rejected by gateway", "status", resp.Status, "error", resp.ErrorMessage)
		payment.Fail(resp.ErrorMessage)
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

		// 8. Publish Collected Events (from Aggregate)
		if len(payment.Events()) > 0 {
			if err := uc.dispatcher.Dispatch(ctx, payment.Events()); err != nil {
				return fmt.Errorf("failed to dispatch events: %w", err)
			}
			payment.ClearEvents()
		}

		return nil
	})

	if err != nil {
		logger.Error(ctx, "Transaction failed", "error", err)
		return nil, nil, err
	}

	// 7. Store Idempotency Result
	if input.IdempotencyKey != "" {
		_ = uc.idempotency.Save(ctx, input.IdempotencyKey, payment)
	}

	metrics.PaymentsTotal.WithLabelValues(order.TenantID, string(payment.Status)).Inc()

	return payment, resp, nil
}

type WebhookPayload struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (uc *PaymentUseCase) HandleWebhook(ctx context.Context, payload WebhookPayload) error {
	
	logger.Info(ctx, "Received webhook from Mercado Pago", 
		"action", payload.Action, 
		"type", payload.Type, 
		"transaction_id", payload.Data.ID)

	if payload.Type != "payment" {
		return nil 
	}

	// 1. Find Payment by TransactionID
	payment, err := uc.paymentRepo.FindByTransactionID(ctx, payload.Data.ID)
	if err != nil {
		return fmt.Errorf("failed to find payment by transaction_id: %w", err)
	}
	if payment == nil {
		logger.Warn(ctx, "Payment not found for webhook transaction", "transaction_id", payload.Data.ID)
		return nil // We don't have this payment, might be from another system or old
	}

	// 2. Get Order and Tenant Config
	order, err := uc.orderRepo.FindByID(ctx, payment.OrderID)
	if err != nil || order == nil {
		return fmt.Errorf("order not found for payment: %w", err)
	}

	tenant, err := uc.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	// 3. Verify Status with Gateway (Security)
	token, _ := uc.getTenantToken(ctx, tenant)
	resp, err := uc.gateway.GetPaymentStatus(ctx, payload.Data.ID, token)
	if err != nil {
		return fmt.Errorf("failed to verify payment status with gateway: %w", err)
	}

	if resp.Status == "approved" {
		logger.Info(ctx, "Payment confirmed via webhook", "transaction_id", payload.Data.ID, "order_id", order.ID)
		
		if err := payment.Approve(); err != nil {
			return err
		}

		// 4. Atomic Update
		err = uc.txManager.Execute(ctx, func(ctx context.Context) error {
			if err := uc.paymentRepo.Save(ctx, payment); err != nil {
				return err
			}
			if err := order.Confirm(); err == nil {
				if err := uc.orderRepo.Save(ctx, order); err != nil {
					return err
				}
			}
			
			// Incrementar uso do plano
			tenant.MonthlyUsageCount++
			if err := uc.tenantRepo.Save(ctx, tenant); err != nil {
				return err
			}

			// 5. Publish Collected Events (from Aggregate)
			if len(payment.Events()) > 0 {
				if err := uc.dispatcher.Dispatch(ctx, payment.Events()); err != nil {
					return fmt.Errorf("failed to dispatch events: %w", err)
				}
				payment.ClearEvents()
			}

			return nil
		})

		if err != nil {
			return err
		}
		
		// Record metrics
		metrics.PaymentsTotal.WithLabelValues(order.TenantID, "approved").Inc()
	}

	return nil
}

// tenantOAuthConfig é a estrutura que armazenamos cifrada no banco.
// Sempre persiste access_token + refresh_token juntos.
type tenantOAuthConfig struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// getTenantToken descriptografa o config do tenant e faz auto-refresh
// se o access_token expirar em menos de 7 dias (margem de segurança).
func (uc *PaymentUseCase) getTenantToken(ctx context.Context, tenant *entity.Tenant) (string, error) {
	if tenant.EncryptedConfig == "" {
		return "", errors.New("tenant has no payment configuration — connect Mercado Pago first")
	}

	decryptedJSON, err := uc.encryption.Decrypt(tenant.EncryptedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt tenant config: %w", err)
	}

	var config tenantOAuthConfig
	if err := json.Unmarshal([]byte(decryptedJSON), &config); err != nil {
		return "", fmt.Errorf("failed to parse tenant config: %w", err)
	}

	// Auto-refresh: se o token expirar em menos de 7 dias, renovar silenciosamente.
	sevenDays := 7 * 24 * time.Hour
	needsRefresh := !tenant.TokenExpiresAt.IsZero() &&
		time.Until(tenant.TokenExpiresAt) < sevenDays &&
		config.RefreshToken != ""

	if needsRefresh {
		logger.Info(ctx, "OAuth token expiring soon, refreshing silently",
			"tenant_id", tenant.ID,
			"expires_at", tenant.TokenExpiresAt)

		newToken, err := uc.oauth.RefreshAccessToken(ctx, config.RefreshToken)
		if err != nil {
			// Falha no refresh não deve bloquear a venda — usamos o token atual
			logger.Warn(ctx, "Silent token refresh failed, using existing token",
				"tenant_id", tenant.ID, "error", err)
		} else {
			// Persistir novos tokens cifrados em background (fire-and-forget)
			go func() {
				newConfig := tenantOAuthConfig{
					AccessToken:  newToken.AccessToken,
					RefreshToken: newToken.RefreshToken,
				}
				newJSON, _ := json.Marshal(newConfig)
				newEncrypted, encErr := uc.encryption.Encrypt(string(newJSON))
				if encErr != nil {
					logger.Warn(ctx, "Failed to encrypt refreshed token", "error", encErr)
					return
				}
				tenant.EncryptedConfig = newEncrypted
				tenant.TokenExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
				_ = uc.tenantRepo.Save(context.Background(), tenant)
				logger.Info(ctx, "OAuth token refreshed and persisted", "tenant_id", tenant.ID)
			}()
			// Usar o novo access_token imediatamente para esta requisição
			config.AccessToken = newToken.AccessToken
		}
	}

	return config.AccessToken, nil
}

// HandleWebhook processa a notificação assíncrona do gateway de pagamento (ex: Mercado Pago).
func (uc *PaymentUseCase) HandleWebhook(ctx context.Context, transactionID string) error {
	// 1. Encontrar o pagamento local usando o ID da transação
	payment, err := uc.paymentRepo.FindByTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("payment not found for transaction %s: %w", transactionID, err)
	}

	// Se já está finalizado, não faz nada (idempotência)
	if payment.Status == entity.PaymentStatusApproved || payment.Status == entity.PaymentStatusFailed {
		logger.Info(ctx, "Webhook ignored, payment already finalized", "payment_id", payment.ID, "status", payment.Status)
		return nil
	}

	// 2. Carregar o Pedido e o Lojista
	order, err := uc.orderRepo.FindByID(ctx, payment.OrderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	tenant, err := uc.tenantRepo.FindByID(ctx, order.TenantID)
	if err != nil {
		return fmt.Errorf("tenant not found: %w", err)
	}

	// 3. Segurança: Perguntar ao Gateway o status real (evita webhook spoofing)
	token, err := uc.getTenantToken(ctx, tenant)
	if err != nil {
		return fmt.Errorf("failed to get tenant token: %w", err)
	}

	gatewayResp, err := uc.gateway.GetPaymentStatus(ctx, transactionID, token)
	if err != nil {
		return fmt.Errorf("failed to verify payment status with gateway: %w", err)
	}

	// 4. Atualizar o status localmente se o Gateway confirmou
	if gatewayResp.Status == "approved" {
		return uc.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
			payment.Status = entity.PaymentStatusApproved
			payment.UpdatedAt = time.Now()
			if err := uc.paymentRepo.Save(txCtx, payment); err != nil {
				return err
			}

			order.Status = entity.OrderConfirmed
			order.UpdatedAt = time.Now()
			if err := uc.orderRepo.Save(txCtx, order); err != nil {
				return err
			}

			logger.Info(ctx, "Payment approved via webhook", "payment_id", payment.ID, "order_id", order.ID)
			return nil
		})
	} else if gatewayResp.Status == "rejected" || gatewayResp.Status == "cancelled" {
		return uc.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
			payment.Status = entity.PaymentStatusFailed
			payment.UpdatedAt = time.Now()
			if err := uc.paymentRepo.Save(txCtx, payment); err != nil {
				return err
			}
			logger.Info(ctx, "Payment rejected via webhook", "payment_id", payment.ID)
			return nil
		})
	}

	return nil
}
