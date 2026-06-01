package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

// --- GET /auth/mercadopago/url ---
// Retorna a URL de autorização do Mercado Pago para o frontend redirecionar o lojista.
// O tenant_id autenticado é colocado no `state` para prevenção de CSRF.
func (h *OrderHandler) MercadoPagoOAuthURL(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(kernel.TenantIDKey).(string)
	if !ok || tenantID == "" {
		RespondWithError(w, r, http.StatusUnauthorized, "tenant not identified")
		return
	}

	// O adapter satisfaz OAuthProvider: gera a URL com o state = tenant_id
	type oauthURLProvider interface {
		OAuthAuthorizationURL(state string) string
	}
	provider, ok := h.mpOAuthProvider.(oauthURLProvider)
	if !ok {
		RespondWithError(w, r, http.StatusInternalServerError, "oauth provider not configured")
		return
	}

	authURL := provider.OAuthAuthorizationURL(tenantID)
	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"url": authURL})
}

// --- GET /auth/mercadopago/callback ---
// Recebido pelo Mercado Pago após o lojista autorizar o aplicativo.
// Troca o `code` temporário por access_token + refresh_token, criptografa e persiste.
func (h *OrderHandler) MercadoPagoOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	tenantID := r.URL.Query().Get("state") // state = tenant_id (anti-CSRF)

	if code == "" || tenantID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "missing code or state")
		return
	}

	// Trocar o code por tokens reais (Server-to-Server, seguro)
	tokens, err := h.mpOAuthProvider.ExchangeOAuthCode(r.Context(), code)
	if err != nil {
		slog.Error("Failed to exchange OAuth code", "tenant_id", tenantID, "error", err)
		http.Redirect(w, r, "/#/settings?status=oauth_error", http.StatusFound)
		return
	}

	// Serializar e criptografar os tokens antes de salvar
	configJSON, _ := json.Marshal(map[string]string{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
	encryptedConfig, err := h.encryptionService.Encrypt(string(configJSON))
	if err != nil {
		slog.Error("Failed to encrypt OAuth tokens", "tenant_id", tenantID, "error", err)
		http.Redirect(w, r, "/#/settings?status=oauth_error", http.StatusFound)
		return
	}

	// Calcular a data de expiração do access_token
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Buscar o tenant e atualizar a configuração de pagamento
	tenant, err := h.tenantRepo.FindByID(r.Context(), tenantID)
	if err != nil || tenant == nil {
		slog.Error("Tenant not found for OAuth callback", "tenant_id", tenantID)
		http.Redirect(w, r, "/#/settings?status=oauth_error", http.StatusFound)
		return
	}

	tenant.EncryptedConfig = encryptedConfig
	tenant.PaymentProvider = "mercado_pago"
	tenant.TokenExpiresAt = expiresAt
	// Marcar o tenant como ativo caso esteja em estado de "onboarding"
	if tenant.Status == "onboarding" {
		tenant.Status = "active"
	}

	if err := h.tenantRepo.Save(r.Context(), tenant); err != nil {
		slog.Error("Failed to save tenant OAuth config", "tenant_id", tenantID, "error", err)
		http.Redirect(w, r, "/#/settings?status=oauth_error", http.StatusFound)
		return
	}

	slog.Info("Mercado Pago OAuth connected successfully",
		"tenant_id", tenantID,
		"mp_user_id", tokens.UserID,
		"expires_at", expiresAt)

	// Redirecionar de volta ao painel com sucesso
	http.Redirect(w, r, "/#/settings?status=connected", http.StatusFound)
}

// mpOAuthConfig é o campo que armazena o adapter OAuth no handler.
// Injetado via SetMPOAuthProvider para evitar dependência circular.
func (h *OrderHandler) SetMPOAuthProvider(provider ports.OAuthProvider) {
	h.mpOAuthProvider = provider
}

// Garante que o Tenant retornado ao banco mantenha os campos calculados
// que não estavam no contrato original de FindByID.
func applyOAuthFieldsToTenant(tenant *entity.Tenant, expiresAt time.Time, encryptedConfig string) {
	tenant.EncryptedConfig = encryptedConfig
	tenant.TokenExpiresAt = expiresAt
}

// Evita erro de "unused import" em caso de refactors futuros
var _ = fmt.Sprintf
