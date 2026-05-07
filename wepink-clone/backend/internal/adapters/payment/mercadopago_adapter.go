package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/awaken-rise/backend/internal/ports"
)

type MercadoPagoAdapter struct {
	client *http.Client
}

func NewMercadoPagoAdapter() *MercadoPagoAdapter {
	return &MercadoPagoAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type mpPaymentRequest struct {
	TransactionAmount float64 `json:"transaction_amount"`
	Token             string  `json:"token,omitempty"`
	Description       string  `json:"description"`
	Installments      int     `json:"installments,omitempty"`
	PaymentMethodID   string  `json:"payment_method_id"`
	Payer             mpPayer `json:"payer"`
}

type mpPayer struct {
	Email string `json:"email"`
}

type mpPaymentResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Detail string `json:"status_detail"`
	Point  struct {
		ExternalResourceURL string `json:"external_resource_url"`
	} `json:"point_of_interaction"`
}

func (a *MercadoPagoAdapter) Process(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentGatewayResponse, error) {
	url := "https://api.mercadopago.com/v1/payments"

	// Mapeamento simplificado para o exemplo
	methodID := "pix"
	if req.PaymentMethod == "credit_card" {
		methodID = "visa" // No mundo real, isso viria do frontend (card_type)
	}

	payload := mpPaymentRequest{
		TransactionAmount: req.Amount,
		Token:             req.Token,
		Description:       req.Description,
		Installments:      1,
		PaymentMethodID:   methodID,
		Payer: mpPayer{
			Email: req.Email,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.TenantToken)
	httpReq.Header.Set("Content-Type", "application/json")
	// Importante: Idempotency-Key para evitar cobrança duplicada
	httpReq.Header.Set("X-Idempotency-Key", fmt.Sprintf("pay-%d", time.Now().UnixNano()))

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return &ports.PaymentGatewayResponse{Success: false, ErrorMessage: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &ports.PaymentGatewayResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Mercado Pago API returned status: %d", resp.StatusCode),
		}, nil
	}

	var mpResp mpPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&mpResp); err != nil {
		return nil, err
	}

	return &ports.PaymentGatewayResponse{
		Success:       mpResp.Status == "approved" || mpResp.Status == "pending",
		TransactionID: fmt.Sprintf("%d", mpResp.ID),
		Status:        mpResp.Status,
		PaymentURL:    mpResp.Point.ExternalResourceURL,
	}, nil
}
