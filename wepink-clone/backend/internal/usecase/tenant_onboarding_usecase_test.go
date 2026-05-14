package usecase

import (
	"context"
	"testing"

	"github.com/awaken-rise/backend/internal/domain/entity"
)

type mockTenantRepo struct {
	tenants map[string]*entity.Tenant
}

func (m *mockTenantRepo) FindByID(ctx context.Context, id string) (*entity.Tenant, error) {
	return m.tenants[id], nil
}

func (m *mockTenantRepo) Save(ctx context.Context, t *entity.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func TestTenantOnboarding_Register(t *testing.T) {
	repo := &mockTenantRepo{tenants: make(map[string]*entity.Tenant)}
	uc := NewTenantOnboardingUseCase(repo)

	t.Run("Register Valid Tenant", func(t *testing.T) {
		input := RegisterTenantInput{
			ID:           "test-id",
			LegalName:    "Test Corp",
			CNPJ:         "12.345.678/0001-00",
			ContactEmail: "test@test.com",
			Plan:         "starter",
		}

		tenant, err := uc.Register(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tenant.CNPJ != "12.345.678/0001-00" {
			t.Errorf("expected CNPJ 12.345.678/0001-00, got %s", tenant.CNPJ)
		}

		if tenant.VerificationStatus != entity.VerificationPending {
			t.Errorf("expected status PENDING, got %s", tenant.VerificationStatus)
		}
	})

	t.Run("Register Invalid CNPJ", func(t *testing.T) {
		input := RegisterTenantInput{
			ID:   "invalid-cnpj",
			CNPJ: "123",
		}

		_, err := uc.Register(context.Background(), input)
		if err == nil {
			t.Error("expected error for invalid CNPJ, got nil")
		}
	})
}
