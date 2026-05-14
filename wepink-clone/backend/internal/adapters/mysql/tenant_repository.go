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
	query := `SELECT tenant_id, legal_name, cnpj, mp_access_token, contact_email, 
	          contact_phone, status, plan_type, verification_status, monthly_usage_count, 
	          created_at, updated_at FROM tenants WHERE tenant_id = ?`
	
	row := r.db.QueryRowContext(ctx, query, id)

	var t entity.Tenant
	err := row.Scan(
		&t.ID, &t.LegalName, &t.CNPJ, &t.MPAccessToken, &t.ContactEmail,
		&t.ContactPhone, &t.Status, &t.Plan, &t.VerificationStatus, 
		&t.MonthlyUsageCount, &t.CreatedAt, &t.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return &t, nil
}

func (r *TenantRepository) Save(ctx context.Context, t *entity.Tenant) error {
	query := `INSERT INTO tenants (
		tenant_id, legal_name, cnpj, mp_access_token, contact_email, 
		contact_phone, status, plan_type, verification_status, monthly_usage_count
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
	ON DUPLICATE KEY UPDATE 
		legal_name = VALUES(legal_name), 
		mp_access_token = VALUES(mp_access_token), 
		status = VALUES(status), 
		plan_type = VALUES(plan_type), 
		verification_status = VALUES(verification_status),
		monthly_usage_count = VALUES(monthly_usage_count)`
	
	_, err := r.db.ExecContext(ctx, query, 
		t.ID, t.LegalName, t.CNPJ, t.MPAccessToken, t.ContactEmail, 
		t.ContactPhone, t.Status, t.Plan, t.VerificationStatus, t.MonthlyUsageCount,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}
	return nil
}
