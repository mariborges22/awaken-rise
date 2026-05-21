package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/awaken-rise/backend/internal/domain/entity"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*entity.Tenant, error) {
	query := `SELECT tenant_id, legal_name, cnpj, payment_provider, encrypted_config, contact_email, 
	          contact_phone, status, plan_type, verification_status, monthly_usage_count, 
	          subscription_id, subscription_status, subscription_expires_at,
	          created_at, updated_at FROM tenants WHERE tenant_id = ?`
	
	row := r.db.QueryRowContext(ctx, query, id)

	var t entity.Tenant
	var encryptedConfigOpt sql.NullString
	var contactPhoneOpt sql.NullString
	var subIDOpt sql.NullString
	var subStatusOpt sql.NullString
	var subExpiresOpt sql.NullTime

	err := row.Scan(
		&t.ID, &t.LegalName, &t.CNPJ, &t.PaymentProvider, &encryptedConfigOpt, &t.ContactEmail,
		&contactPhoneOpt, &t.Status, &t.Plan, &t.VerificationStatus, 
		&t.MonthlyUsageCount, &subIDOpt, &subStatusOpt, &subExpiresOpt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	t.EncryptedConfig = encryptedConfigOpt.String
	t.ContactPhone = contactPhoneOpt.String
	t.SubscriptionID = subIDOpt.String
	t.SubscriptionStatus = subStatusOpt.String
	if subExpiresOpt.Valid {
		t.SubscriptionExpiresAt = subExpiresOpt.Time
	}

	return &t, nil
}

func (r *TenantRepository) Save(ctx context.Context, t *entity.Tenant) error {
	query := `INSERT INTO tenants (
		tenant_id, legal_name, cnpj, payment_provider, encrypted_config, contact_email, 
		contact_phone, status, plan_type, verification_status, monthly_usage_count,
		subscription_id, subscription_status, subscription_expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
	ON DUPLICATE KEY UPDATE 
		legal_name = VALUES(legal_name), 
		payment_provider = VALUES(payment_provider),
		encrypted_config = VALUES(encrypted_config),
		status = VALUES(status), 
		plan_type = VALUES(plan_type), 
		verification_status = VALUES(verification_status),
		monthly_usage_count = VALUES(monthly_usage_count),
		subscription_id = VALUES(subscription_id),
		subscription_status = VALUES(subscription_status),
		subscription_expires_at = VALUES(subscription_expires_at)`
	
	var expiresVal interface{}
	if !t.SubscriptionExpiresAt.IsZero() {
		expiresVal = t.SubscriptionExpiresAt
	} else {
		expiresVal = nil
	}

	var configVal interface{}
	if t.EncryptedConfig != "" {
		configVal = t.EncryptedConfig
	} else {
		configVal = nil
	}

	var phoneVal interface{}
	if t.ContactPhone != "" {
		phoneVal = t.ContactPhone
	} else {
		phoneVal = nil
	}

	_, err := r.db.ExecContext(ctx, query, 
		t.ID, t.LegalName, t.CNPJ, t.PaymentProvider, configVal, t.ContactEmail, 
		phoneVal, t.Status, t.Plan, t.VerificationStatus, t.MonthlyUsageCount,
		t.SubscriptionID, t.SubscriptionStatus, expiresVal,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}
	return nil
}
