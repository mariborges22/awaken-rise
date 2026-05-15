package http

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/ports"
	"github.com/awaken-rise/backend/internal/usecase"
)

type OrderHandler struct {
	orderUseCase     *usecase.OrderUseCase
	paymentUseCase   *usecase.PaymentUseCase
	db               *sql.DB
	tenantRepo       ports.TenantRepository
	tenantOnboarding *usecase.TenantOnboardingUseCase
	publisher        ports.EventPublisher
	encryption       *service.EncryptionService
}

func NewOrderHandler(
	orderUseCase *usecase.OrderUseCase, 
	paymentUseCase *usecase.PaymentUseCase,
	db *sql.DB,
	tenantRepo ports.TenantRepository,
	publisher ports.EventPublisher,
	tenantOnboarding *usecase.TenantOnboardingUseCase,
) *OrderHandler {
	return &OrderHandler{
		orderUseCase:     orderUseCase,
		paymentUseCase:   paymentUseCase,
		db:               db,
		tenantRepo:       tenantRepo,
		publisher:        publisher,
		tenantOnboarding: tenantOnboarding,
	}
}

func (h *OrderHandler) SetEncryptionService(s *service.EncryptionService) {
	h.encryption = s
}

type CreateOrderRequest struct {
	TenantID string             `json:"tenant_id"`
	Items    []entity.OrderItem `json:"items"`
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TenantID == "" {
		req.TenantID = "default-tenant" // Default fallback
	}

	order, err := h.orderUseCase.CreateOrder(r.Context(), uuid.New().String(), req.TenantID, req.Items)
	if err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	RespondWithSuccess(w, r, http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		RespondWithError(w, r, http.StatusBadRequest, "Missing order id")
		return
	}

	order, err := h.orderUseCase.GetOrder(r.Context(), id)
	if err != nil {
		RespondWithError(w, r, http.StatusNotFound, "Order not found")
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
		RespondWithError(w, r, http.StatusBadRequest, "Missing order id")
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
		RespondWithError(w, r, http.StatusInternalServerError, err.Error())
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

	if h.publisher == nil {
		RespondWithError(w, r, http.StatusServiceUnavailable, "RabbitMQ not ready")
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *OrderHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var payload usecase.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid webhook payload")
		return
	}

	// Security Note: Validate X-Signature or x-correlation-id headers here based on MercadoPago docs

	if err := h.paymentUseCase.HandleWebhook(r.Context(), payload); err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, err.Error())
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
		RespondWithError(w, r, http.StatusBadRequest, err.Error())
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

	// 1. Recuperar Tenant (Em produção viria do Contexto/JWT)
	// Para o MVP, estamos assumindo o fluxo de teste
	tenantID := "default-tenant" 
	tenant, err := h.tenantRepo.FindByID(r.Context(), tenantID)
	if err != nil || tenant == nil {
		RespondWithError(w, r, http.StatusNotFound, "Tenant not found")
		return
	}

	// 2. Serializar Configurações
	settingsJSON, _ := json.Marshal(req.Settings)

	// 3. Criptografar
	encrypted, err := h.encryption.Encrypt(string(settingsJSON))
	if err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to secure settings")
		return
	}

	// 4. Salvar
	tenant.PaymentProvider = req.Provider
	tenant.EncryptedConfig = encrypted

	if err := h.tenantRepo.Save(r.Context(), tenant); err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to save config")
		return
	}

	RespondWithSuccess(w, r, http.StatusOK, map[string]string{"status": "configured_and_encrypted"})
}

func (h *OrderHandler) Metrics() http.Handler {
	return promhttp.Handler()
}

