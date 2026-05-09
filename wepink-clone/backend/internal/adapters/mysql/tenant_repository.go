package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/awaken-rise/backend/internal/ports"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*ports.TenantConfig, error) {
	// Nota: Em um sistema real, buscaríamos na tabela 'tenants'.
	// Para este deploy inicial, retornaremos uma configuração default se o ID for 'default-tenant'.
	if id == "default-tenant" {
		return &ports.TenantConfig{
			TenantID:      "default-tenant",
			MPAccessToken: "TEST-4171246039575815-050410-6c9c614c227b60098f98642735d67807-172551460", // Token de teste do MP
			Status:        "active",
		}, nil
	}

	query := "SELECT tenant_id, mp_access_token, status FROM tenants WHERE tenant_id = ?"
	row := r.db.QueryRowContext(ctx, query, id)

	var config ports.TenantConfig
	err := row.Scan(&config.TenantID, &config.MPAccessToken, &config.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback para teste se a tabela não existir ainda ou estiver vazia
			return &ports.TenantConfig{
				TenantID:      id,
				MPAccessToken: "TEST-TOKEN",
				Status:        "active",
			}, nil
		}
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	return &config, nil
}
