package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// OAuthTokenResponse é a resposta do endpoint /oauth/token do Mercado Pago.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	// ExpiresIn em segundos (normalmente 15552000 = 180 dias para access_token)
	ExpiresIn int `json:"expires_in"`
	// UserID é o ID do lojista no Mercado Pago (útil para logs e associação)
	UserID    int64  `json:"user_id"`
	TokenType string `json:"token_type"`
}

// OAuthAuthorizationURL monta a URL de autorização para redirecionar o lojista.
// O parâmetro `state` deve conter o tenant_id para prevenção de CSRF.
func (a *MercadoPagoAdapter) OAuthAuthorizationURL(state string) string {
	appID := os.Getenv("MP_APP_ID")
	redirectURI := os.Getenv("MP_REDIRECT_URI")
	return fmt.Sprintf(
		"https://auth.mercadopago.com/authorization?client_id=%s&response_type=code&platform_id=mp&redirect_uri=%s&state=%s",
		appID, url.QueryEscape(redirectURI), url.QueryEscape(state),
	)
}

// ExchangeOAuthCode troca o `code` temporário (retornado no callback) por
// um access_token + refresh_token de longa duração via Server-to-Server.
func (a *MercadoPagoAdapter) ExchangeOAuthCode(ctx context.Context, code string) (*ports.OAuthTokenResult, error) {
	appID := os.Getenv("MP_APP_ID")
	clientSecret := os.Getenv("MP_CLIENT_SECRET")
	redirectURI := os.Getenv("MP_REDIRECT_URI")

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", appID)
	params.Set("client_secret", clientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", redirectURI)

	raw, err := a.requestToken(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ports.OAuthTokenResult{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresIn:    raw.ExpiresIn,
		UserID:       raw.UserID,
	}, nil
}

// RefreshAccessToken renova silenciosamente um access_token expirado usando o refresh_token.
// Deve ser chamado pelo PaymentUseCase quando a API do MP retornar HTTP 401.
func (a *MercadoPagoAdapter) RefreshAccessToken(ctx context.Context, refreshToken string) (*ports.OAuthTokenResult, error) {
	appID := os.Getenv("MP_APP_ID")
	clientSecret := os.Getenv("MP_CLIENT_SECRET")

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("client_id", appID)
	params.Set("client_secret", clientSecret)
	params.Set("refresh_token", refreshToken)

	raw, err := a.requestToken(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ports.OAuthTokenResult{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresIn:    raw.ExpiresIn,
		UserID:       raw.UserID,
	}, nil
}

// requestToken é o helper compartilhado que faz o POST /oauth/token.
func (a *MercadoPagoAdapter) requestToken(ctx context.Context, params url.Values) (*OAuthTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mercadopago.com/oauth/token",
		strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var mpErr map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&mpErr)
		return nil, fmt.Errorf("MP OAuth error %d: %v", resp.StatusCode, mpErr["message"])
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	return &tokenResp, nil
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
			TicketURL    string `json:"ticket_url"`
			QRCode       string `json:"qr_code"`
			QRCodeBase64 string `json:"qr_code_base64"`
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
		Success:         mpResp.Status == "approved" || mpResp.Status == "pending",
		TransactionID:   fmt.Sprintf("%d", mpResp.ID),
		Status:          mpResp.Status,
		PaymentURL:      mpResp.PointOfInteraction.TransactionData.TicketURL,
		PixQRCodeBase64: mpResp.PointOfInteraction.TransactionData.QRCodeBase64,
		PixCopyPaste:    mpResp.PointOfInteraction.TransactionData.QRCode,
		ErrorMessage:    mpResp.StatusDetail,
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
