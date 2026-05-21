package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/awaken-rise/backend/internal/ports"
)

type BillingUseCase struct {
	tenantRepo ports.TenantRepository
}

func NewBillingUseCase(tenantRepo ports.TenantRepository) *BillingUseCase {
	return &BillingUseCase{tenantRepo: tenantRepo}
}

type BillingWebhookInput struct {
	SubscriptionID string    `json:"subscription_id"`
	TenantID       string    `json:"tenant_id"`
	Status         string    `json:"status"` // "active", "cancelled", "suspended", "trialing"
	ExpiresAt      time.Time `json:"expires_at"`
}

func (uc *BillingUseCase) HandleSubscriptionUpdate(ctx context.Context, input BillingWebhookInput) error {
	tenant, err := uc.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		return fmt.Errorf("failed to query tenant: %w", err)
	}
	if tenant == nil {
		return errors.New("tenant not found for billing update")
	}

	tenant.SubscriptionID = input.SubscriptionID
	tenant.SubscriptionStatus = input.Status
	tenant.SubscriptionExpiresAt = input.ExpiresAt
	tenant.UpdatedAt = time.Now()

	if err := uc.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant subscription update: %w", err)
	}

	return nil
}
