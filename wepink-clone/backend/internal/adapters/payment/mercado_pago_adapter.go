package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type MercadoPagoAdapter struct {
	httpClient *http.Client
}

func NewMercadoPagoAdapter() *MercadoPagoAdapter {
	return &MercadoPagoAdapter{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type MPPaymentRequest struct {
	TransactionAmount kernel.Money `json:"transaction_amount"`
	Description       string       `json:"description"`
	PaymentMethodID   string       `json:"payment_method_id"`
	Payer             MPPayer      `json:"payer"`
	Token             string       `json:"token,omitempty"`
	Installments      int          `json:"installments,omitempty"`
}

type MPPayer struct {
	Email string `json:"email"`
}

type MPPaymentResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	PointOfInteraction struct {
		TransactionData struct {
			TicketURL string `json:"ticket_url"`
			QRCode    string `json:"qr_code"`
		} `json:"transaction_data"`
	} `json:"point_of_interaction"`
	StatusDetail string `json:"status_detail"`
}

func (a *MercadoPagoAdapter) Process(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentGatewayResponse, error) {
	// Se não houver token do lojista, não podemos processar
	if req.TenantToken == "" {
		return nil, fmt.Errorf("merchant access token is required")
	}

	mpReq := MPPaymentRequest{
		TransactionAmount: req.Amount,
		Description:       req.Description,
		PaymentMethodID:   req.PaymentMethod,
		Payer: MPPayer{
			Email: req.Email,
		},
		Token:        req.Token,
		Installments: 1, // Por padrão, mas pode ser parametrizado
	}

	body, _ := json.Marshal(mpReq)
	
	url := "https://api.mercadopago.com/v1/payments"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+req.TenantToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Idempotency-Key", fmt.Sprintf("awaken-%d", time.Now().UnixNano()))

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var mpErr map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&mpErr)
		return &ports.PaymentGatewayResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("MP Error: %v", mpErr["message"]),
		}, nil
	}

	var mpResp MPPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&mpResp); err != nil {
		return nil, err
	}

	return &ports.PaymentGatewayResponse{
		Success:       mpResp.Status == "approved" || mpResp.Status == "pending",
		TransactionID: fmt.Sprintf("%d", mpResp.ID),
		Status:        mpResp.Status,
		PaymentURL:    mpResp.PointOfInteraction.TransactionData.TicketURL,
		ErrorMessage:  mpResp.StatusDetail,
	}, nil
}

func (a *MercadoPagoAdapter) GetPaymentStatus(ctx context.Context, transactionID string, tenantToken string) (*ports.PaymentGatewayResponse, error) {
	url := fmt.Sprintf("https://api.mercadopago.com/v1/payments/%s", transactionID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+tenantToken)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &ports.PaymentGatewayResponse{
			Success:      false,
			ErrorMessage: "Failed to fetch payment status",
		}, nil
	}

	var mpResp MPPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&mpResp); err != nil {
		return nil, err
	}

	return &ports.PaymentGatewayResponse{
		Success:       mpResp.Status == "approved",
		TransactionID: fmt.Sprintf("%d", mpResp.ID),
		Status:        mpResp.Status,
		ErrorMessage:  mpResp.StatusDetail,
	}, nil
}
