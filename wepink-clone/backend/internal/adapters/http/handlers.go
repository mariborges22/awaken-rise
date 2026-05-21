package http

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/awaken-rise/backend/internal/domain/service"
	"github.com/awaken-rise/backend/internal/ports"
	"github.com/awaken-rise/backend/internal/usecase"
)

type OrderHandler struct {
	orderUseCase     *usecase.OrderUseCase
	paymentUseCase   *usecase.PaymentUseCase
	db               *sql.DB
	tenantRepo       ports.TenantRepository
	tenantOnboarding *usecase.TenantOnboardingUseCase
	dispatcher       ports.EventDispatcher
	encryption       *service.EncryptionService
	authUseCase      *usecase.AuthUseCase
	productUseCase   *usecase.ProductUseCase
	billingUseCase   *usecase.BillingUseCase
}

func NewOrderHandler(
	orderUseCase *usecase.OrderUseCase, 
	paymentUseCase *usecase.PaymentUseCase,
	db *sql.DB,
	tenantRepo ports.TenantRepository,
	dispatcher ports.EventDispatcher,
	tenantOnboarding *usecase.TenantOnboardingUseCase,
) *OrderHandler {
	return &OrderHandler{
		orderUseCase:     orderUseCase,
		paymentUseCase:   paymentUseCase,
		db:               db,
		tenantRepo:       tenantRepo,
		dispatcher:       dispatcher,
		tenantOnboarding: tenantOnboarding,
	}
}

func (h *OrderHandler) SetEncryptionService(s *service.EncryptionService) {
	h.encryption = s
}

func (h *OrderHandler) SetAuthUseCase(s *usecase.AuthUseCase) {
	h.authUseCase = s
}

func (h *OrderHandler) SetProductUseCase(s *usecase.ProductUseCase) {
	h.productUseCase = s
}

func (h *OrderHandler) SetBillingUseCase(s *usecase.BillingUseCase) {
	h.billingUseCase = s
}

type RegisterUserRequest struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *OrderHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	input := usecase.RegisterUserInput{
		TenantID: req.TenantID,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     "admin",
	}

	user, err := h.authUseCase.Register(r.Context(), input)
	if err != nil {
		RespondWithError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	RespondWithSuccess(w, r, http.StatusCreated, user)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *OrderHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	input := usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	out, err := h.authUseCase.Login(r.Context(), input)
	if err != nil {
		RespondWithError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, out)
}

type CreateOrderRequest struct {
	TenantID string                         `json:"tenant_id"`
	Items    []usecase.CreateOrderInputItem `json:"items"`
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	if req.TenantID != "" {
		ctx = kernel.WithTenantID(ctx, req.TenantID)
	}

	order, err := h.orderUseCase.CreateOrder(ctx, uuid.New().String(), req.Items)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusCreated, order)
}

func (h *OrderHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	product, err := h.productUseCase.CreateProduct(r.Context(), req)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusCreated, product)
}

func (h *OrderHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productUseCase.ListProducts(r.Context())
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, products)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		HandleError(w, r, fmt.Errorf("%w: missing order id", kernel.ErrInvalidInput))
		return
	}

	order, err := h.orderUseCase.GetOrder(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, order)
}

type ProcessPaymentRequest struct {
	PaymentMethod string `json:"payment_method"`
	CardToken     string `json:"card_token"`
	BuyerEmail    string `json:"buyer_email"`
}

func (h *OrderHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("orderId")
	if orderID == "" {
		HandleError(w, r, fmt.Errorf("%w: missing order id", kernel.ErrInvalidInput))
		return
	}

	var req ProcessPaymentRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error, use defaults if empty

	input := usecase.ProcessPaymentInput{
		PaymentID:      uuid.New().String(),
		OrderID:        orderID,
		IdempotencyKey: r.Header.Get("X-Idempotency-Key"),
		PaymentMethod:  req.PaymentMethod,
		CardToken:      req.CardToken,
		BuyerEmail:     req.BuyerEmail,
	}

	if input.IdempotencyKey == "" {
		input.IdempotencyKey = uuid.New().String()
	}

	payment, err := h.paymentUseCase.ProcessPayment(r.Context(), input)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, payment)
}

func (h *OrderHandler) Live(w http.ResponseWriter, r *http.Request) {
	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "up"})
}

func (h *OrderHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		RespondWithError(w, r, http.StatusServiceUnavailable, "MySQL not ready")
		return
	}

	if h.dispatcher == nil {
		RespondWithError(w, r, http.StatusServiceUnavailable, "RabbitMQ Dispatcher not ready")
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *OrderHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var payload usecase.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, r, fmt.Errorf("%w: invalid webhook payload", kernel.ErrInvalidInput))
		return
	}

	// Security Note: Validate X-Signature or x-correlation-id headers here based on MercadoPago docs

	if err := h.paymentUseCase.HandleWebhook(r.Context(), payload); err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "received"})
}

type RegisterTenantRequest struct {
	TenantID      string `json:"tenant_id"`
	LegalName     string `json:"legal_name"`
	CNPJ          string `json:"cnpj"`
	ContactEmail  string `json:"contact_email"`
	Plan          string `json:"plan"`
}

func (h *OrderHandler) RegisterTenant(w http.ResponseWriter, r *http.Request) {
	var req RegisterTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	input := usecase.RegisterTenantInput{
		ID:            req.TenantID,
		LegalName:     req.LegalName,
		CNPJ:          req.CNPJ,
		ContactEmail:  req.ContactEmail,
		Plan:          req.Plan,
	}

	tenant, err := h.tenantOnboarding.Register(r.Context(), input)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusCreated, tenant)
}

type UpdateTenantConfigRequest struct {
	Provider string                 `json:"provider"`
	Settings map[string]interface{} `json:"settings"`
}

func (h *OrderHandler) UpdateTenantConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateTenantConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Recuperar Tenant do Contexto (X-Tenant-ID header) com fallback seguro
	tenantID, ok := kernel.GetTenantID(r.Context())
	if !ok || tenantID == "" {
		tenantID = "default-tenant"
	}
	tenant, err := h.tenantRepo.FindByID(r.Context(), tenantID)
	if err != nil || tenant == nil {
		HandleError(w, r, fmt.Errorf("%w: tenant not found", kernel.ErrNotFound))
		return
	}

	// 2. Serializar Configurações
	settingsJSON, _ := json.Marshal(req.Settings)

	// 3. Criptografar
	encrypted, err := h.encryption.Encrypt(string(settingsJSON))
	if err != nil {
		HandleError(w, r, fmt.Errorf("%w: failed to secure settings", kernel.ErrInternal))
		return
	}

	// 4. Salvar
	tenant.PaymentProvider = req.Provider
	tenant.EncryptedConfig = encrypted

	if err := h.tenantRepo.Save(r.Context(), tenant); err != nil {
		HandleError(w, r, fmt.Errorf("%w: failed to save config", kernel.ErrInternal))
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "configured_and_encrypted"})
}

func (h *OrderHandler) GetTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := kernel.GetTenantID(r.Context())
	if !ok || tenantID == "" {
		tenantID = "default-tenant"
	}

	tenant, err := h.tenantRepo.FindByID(r.Context(), tenantID)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	if tenant == nil {
		RespondWithError(w, r, http.StatusNotFound, "Tenant not found")
		return
	}

	var settings map[string]interface{}
	if tenant.EncryptedConfig != "" {
		if h.encryption == nil {
			HandleError(w, r, fmt.Errorf("%w: encryption service not initialized", kernel.ErrInternal))
			return
		}
		
		decrypted, err := h.encryption.Decrypt(tenant.EncryptedConfig)
		if err != nil {
			HandleError(w, r, fmt.Errorf("%w: failed to decrypt settings", kernel.ErrInternal))
			return
		}

		if err := json.Unmarshal([]byte(decrypted), &settings); err != nil {
			HandleError(w, r, fmt.Errorf("%w: failed to parse settings", kernel.ErrInternal))
			return
		}

		// Mascaramento de dados sensíveis para segurança e compliance
		for k, v := range settings {
			if k == "access_token" {
				if strVal, ok := v.(string); ok && len(strVal) > 10 {
					settings[k] = strVal[:8] + "..." + strVal[len(strVal)-4:]
				} else {
					settings[k] = "******"
				}
			}
		}
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]interface{}{
		"provider": tenant.PaymentProvider,
		"settings": settings,
	})
}

func (h *OrderHandler) Metrics() http.Handler {
	return promhttp.Handler()
}

func (h *OrderHandler) BillingWebhook(w http.ResponseWriter, r *http.Request) {
	var req usecase.BillingWebhookInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TenantID == "" {
		RespondWithError(w, r, http.StatusBadRequest, "Missing tenant_id")
		return
	}

	if err := h.billingUseCase.HandleSubscriptionUpdate(r.Context(), req); err != nil {
		HandleError(w, r, err)
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "billing_updated"})
}


