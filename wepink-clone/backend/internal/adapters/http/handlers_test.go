package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/domain/service"
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

func TestOrderHandler_TenantConfig(t *testing.T) {
	repo := &mockTenantRepo{tenants: make(map[string]*entity.Tenant)}
	encService, _ := service.NewEncryptionService("12345678901234567890123456789012") // 32 bytes key

	handler := NewOrderHandler(nil, nil, nil, repo, nil, nil)
	handler.SetEncryptionService(encService)

	// Seed default-tenant
	repo.tenants["default-tenant"] = &entity.Tenant{
		ID:              "default-tenant",
		LegalName:       "Default Corp",
		CNPJ:            "00000000000000",
		PaymentProvider: "mercadopago",
		EncryptedConfig: "",
	}

	// Seed tnt-123
	repo.tenants["tnt-123"] = &entity.Tenant{
		ID:              "tnt-123",
		LegalName:       "Custom Corp",
		CNPJ:            "11111111111111",
		PaymentProvider: "",
		EncryptedConfig: "",
	}

	t.Run("UpdateTenantConfig - Success with Dynamic Tenant ID", func(t *testing.T) {
		reqBody := `{"provider":"mercadopago","settings":{"access_token":"APP_USR-1234567890123456"}}`
		req := httptest.NewRequest("PUT", "/tenants/me/config", bytes.NewBufferString(reqBody))
		
		// Injetar tenant ID no contexto
		ctx := kernel.WithTenantID(req.Context(), "tnt-123")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.UpdateTenantConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rr.Code)
		}

		// Verificar se foi salvo no repo do tnt-123
		tenant := repo.tenants["tnt-123"]
		if tenant.PaymentProvider != "mercadopago" {
			t.Errorf("expected provider mercadopago, got %s", tenant.PaymentProvider)
		}

		if tenant.EncryptedConfig == "" {
			t.Error("expected config to be encrypted, got empty string")
		}
	})

	t.Run("UpdateTenantConfig - Success with Fallback to default-tenant", func(t *testing.T) {
		reqBody := `{"provider":"mercadopago","settings":{"access_token":"APP_USR-fallback"}}`
		req := httptest.NewRequest("PUT", "/tenants/me/config", bytes.NewBufferString(reqBody))
		// Sem injetar tenant ID no contexto (simula requisição sem header)

		rr := httptest.NewRecorder()

		handler.UpdateTenantConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rr.Code)
		}

		tenant := repo.tenants["default-tenant"]
		if tenant.PaymentProvider != "mercadopago" {
			t.Errorf("expected provider mercadopago, got %s", tenant.PaymentProvider)
		}
	})

	t.Run("GetTenantConfig - Success and Masks Access Token", func(t *testing.T) {
		// Primeiro encriptar e salvar um token no tnt-123
		settingsJSON := `{"access_token":"APP_USR-9876543210987654321"}`
		encrypted, _ := encService.Encrypt(settingsJSON)
		
		repo.tenants["tnt-123"] = &entity.Tenant{
			ID:              "tnt-123",
			PaymentProvider: "mercadopago",
			EncryptedConfig: encrypted,
		}

		req := httptest.NewRequest("GET", "/tenants/me/config", nil)
		ctx := kernel.WithTenantID(req.Context(), "tnt-123")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetTenantConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)

		// Verificar resposta envelopada (RespondWithSuccess usa "data")
		data := resp["data"].(map[string]interface{})
		provider := data["provider"].(string)
		settings := data["settings"].(map[string]interface{})
		accessToken := settings["access_token"].(string)

		if provider != "mercadopago" {
			t.Errorf("expected provider mercadopago, got %s", provider)
		}

		// O token deve estar mascarado (ex: APP_USR-...654321 ou similar)
		if !strings.Contains(accessToken, "...") {
			t.Errorf("expected token to be masked with '...', got %s", accessToken)
		}
		
		if strings.Contains(accessToken, "9876543210") {
			t.Errorf("security leak: original token visible in response: %s", accessToken)
		}
	})

	t.Run("GetTenantConfig - Tenant Not Found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tenants/me/config", nil)
		ctx := kernel.WithTenantID(req.Context(), "non-existent")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetTenantConfig(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rr.Code)
		}
	})
}
