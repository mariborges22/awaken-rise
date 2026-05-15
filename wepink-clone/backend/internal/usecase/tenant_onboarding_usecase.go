package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/ports"
)

type TenantOnboardingUseCase struct {
	repo ports.TenantRepository
}

func NewTenantOnboardingUseCase(repo ports.TenantRepository) *TenantOnboardingUseCase {
	return &TenantOnboardingUseCase{repo: repo}
}

type RegisterTenantInput struct {
	ID            string
	LegalName     string
	CNPJ          string
	ContactEmail  string
	Plan          string
}

func (uc *TenantOnboardingUseCase) Register(ctx context.Context, input RegisterTenantInput) (*entity.Tenant, error) {
	// 1. Validação Básica de Negócio
	if !uc.validateCNPJ(input.CNPJ) {
		return nil, errors.New("invalid or irregular CNPJ")
	}

	// 2. Verificar se já existe
	existing, _ := uc.repo.FindByID(ctx, input.ID)
	if existing != nil {
		return nil, fmt.Errorf("tenant already exists with ID: %s", input.ID)
	}

	// 3. Criar Entidade com Status Pendente (Segurança)
	tenant := &entity.Tenant{
		ID:                 input.ID,
		LegalName:          input.LegalName,
		CNPJ:               input.CNPJ,
		ContactEmail:       input.ContactEmail,
		Status:             "active",
		Plan:               entity.PlanType(strings.ToLower(input.Plan)),
		VerificationStatus: entity.VerificationPending,
	}

	if err := uc.repo.Save(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to register tenant: %w", err)
	}

	slog.Info("New tenant registered, awaiting verification", "tenant_id", tenant.ID, "cnpj", tenant.CNPJ)
	
	return tenant, nil
}

func (uc *TenantOnboardingUseCase) Approve(ctx context.Context, tenantID string) error {
	tenant, err := uc.repo.FindByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return errors.New("tenant not found")
	}

	tenant.VerificationStatus = entity.VerificationApproved
	
	slog.Info("Tenant approved for sales", "tenant_id", tenantID)
	return uc.repo.Save(ctx, tenant)
}

func (uc *TenantOnboardingUseCase) validateCNPJ(cnpj string) bool {
	// Aqui entrará a integração com API da Receita Federal ou validador real.
	// Por enquanto, garantimos que tem o tamanho correto e não está vazio.
	cleanCNPJ := strings.ReplaceAll(strings.ReplaceAll(cnpj, ".", ""), "/", "")
	cleanCNPJ = strings.ReplaceAll(cleanCNPJ, "-", "")
	
	return len(cleanCNPJ) == 14
}
